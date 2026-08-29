package gen

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ParseFile extracts the components defined in one JavaScript source file.
func ParseFile(path string) ([]Component, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSource(filepath.ToSlash(path), string(src))
}

// ParseDir walks a directory tree and extracts every component it finds in
// .js and .mjs files. Directories named node_modules, dist and .git are
// skipped, as is anything starting with a dot.
func ParseDir(root string) ([]Component, error) {
	var out []Component
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "dist" || name == "vendor") {
				return fs.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(name)
		if ext != ".js" && ext != ".mjs" {
			return nil
		}
		if strings.HasSuffix(name, ".test.js") || strings.HasSuffix(name, ".min.js") {
			return nil
		}
		components, err := ParseFile(path)
		if err != nil {
			return err
		}
		out = append(out, components...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ParseSource extracts components from JavaScript source. name is used in
// error messages and in the Source field.
func ParseSource(name, src string) ([]Component, error) {
	p, err := newParser(name, src)
	if err != nil {
		return nil, err
	}

	var (
		out        []Component
		docTok     token
		hasDoc     bool
		documented = map[string]bool{}
	)

	for i := 0; i < len(p.toks); i++ {
		t := p.toks[i]

		switch {
		case t.kind == tDoc:
			docTok, hasDoc = t, true
			continue

		case t.kind == tPunct && (t.val == ";" || t.val == "}"):
			// A doc comment describes what comes straight after it. Once a
			// statement has ended, it belongs to something else.
			hasDoc = false
			continue

		case t.kind != tIdent || t.val != "define" || !p.at(i+1).is("("):
			continue
		}

		// customElements.define registers a class, not an alacris component.
		if p.at(i-1).is(".") && p.at(i-2).kind == tIdent && p.at(i-2).val == "customElements" {
			hasDoc = false
			continue
		}

		p.pos = i + 2 // past `define` and `(`
		c, ok, err := p.parseDefine(name)
		if err != nil {
			return nil, err
		}
		if ok {
			c.Source = fmt.Sprintf("%s:%d", name, t.line)
			if hasDoc {
				doc := parseDoc(docTok.val, docTok.line)
				if err := applyDoc(&c, doc); err != nil {
					var u unknownPropError
					if errors.As(err, &u) {
						if tag, line, found := suggestDocOwner(p, i, doc); found {
							return nil, fmt.Errorf("%s:%d: doc block documents props not on <%s>; did you mean <%s> at line %d?", name, docTok.line, c.Tag, tag, line)
						}
					}
					return nil, fmt.Errorf("%s:%d: <%s>: %w", name, docTok.line, c.Tag, err)
				}
				documented[c.Tag] = true
			}
			out = append(out, c)
		}
		// Resume after the call rather than inside it, so a nested define is
		// not read twice.
		if p.pos > i {
			i = p.pos - 1
		}
		hasDoc = false
	}

	// Alacris UI (and anything that follows its conventions) puts the
	// component's documentation in a // block at the top of the file, then
	// imports and helpers, then define(). That block is not "straight after"
	// the call, so the usual association misses it. One define() per file is
	// the rule those sources follow; with more than one, which component the
	// header describes would be a guess.
	if leading, ok := leadingFileDoc(src); ok && len(out) == 1 && !documented[out[0].Tag] {
		if err := applyDoc(&out[0], leading); err != nil {
			return nil, fmt.Errorf("%s: <%s>: %w", name, out[0].Tag, err)
		}
	}

	applyVarsCalls(src, out)

	for i := range out {
		if err := inferProps(&out[i]); err != nil {
			return nil, fmt.Errorf("%s: <%s>: %w", out[i].Source, out[i].Tag, err)
		}
	}

	return out, nil
}

// rawProp carries a prop's default until JSDoc has had its say.
type rawProp struct {
	name string
	val  jsValue
	doc  string
}

// parseDefine reads one define(...) call, with the parser positioned just
// after the opening parenthesis. ok is false for a call that is not an alacris
// component definition.
func (p *parser) parseDefine(file string) (c Component, ok bool, err error) {
	tag, err := p.parseValue()
	if err != nil {
		return c, false, err
	}
	// A tag built at runtime cannot be registered as a custom element name we
	// can write into Go, and a non-string first argument is some other
	// define() — AMD, most likely.
	if tag.kind != jsString || !strings.Contains(tag.str, "-") {
		return c, false, nil
	}
	c.Tag = tag.str

	if !p.cur().is(",") {
		return c, false, nil
	}
	p.advance()

	opts, err := p.parseValue()
	if err != nil {
		return c, false, err
	}

	switch opts.kind {
	case jsObject:
		if !isDefineOptions(opts) {
			return c, false, nil
		}
	case jsOpaque:
		// define('x-hello', () => html`...`) — the shorthand, no props.
	default:
		return c, false, nil
	}

	if shadow, found := opts.get("shadow"); found {
		switch {
		case shadow.kind == jsString:
			c.Shadow = shadow.str
		case shadow.kind == jsBool && !shadow.boolean:
			c.Shadow = "none"
		}
	}

	props, found := opts.get("props")
	if !found {
		return c, true, nil
	}
	if props.kind != jsObject {
		return c, false, fmt.Errorf("%s:%d: <%s>: props is %s, not an object literal; declare the props in a manifest instead",
			file, props.line, c.Tag, props.kind)
	}
	if props.partial {
		// Some of the props are contributed by something the scanner cannot
		// see — a spread, a computed key. Generating the ones it can read
		// would produce a type that is quietly missing fields.
		return c, false, fmt.Errorf("%s:%d: <%s>: props holds a spread or a computed key, so the full set of props is not visible here; declare this component in a manifest instead",
			file, props.line, c.Tag)
	}

	for _, f := range props.obj {
		prop := Prop{Name: f.key, Attr: attrName(f.key)}
		if f.doc != "" {
			prop.Doc = parseDoc(f.doc, f.line).description
		}
		c.Props = append(c.Props, prop)
		c.rawProps = append(c.rawProps, rawProp{name: f.key, val: f.val, doc: f.doc})
	}
	return c, true, nil
}

// isDefineOptions distinguishes alacris' options object from any other object
// that happens to be a second argument.
func isDefineOptions(v jsValue) bool {
	for _, key := range []string{"props", "setup", "styles", "shadow"} {
		if v.hasKey(key) {
			return true
		}
	}
	return false
}

// applyDoc folds a JSDoc block into the component.
func applyDoc(c *Component, doc docBlock) error {
	c.Doc = doc.description

	if t, ok := doc.find("goname"); ok {
		c.GoName = strings.Fields(t.text + " ")[0]
	}
	if t, ok := doc.find("deprecated"); ok {
		c.Deprecated = t.text
		if c.Deprecated == "" {
			c.Deprecated = "this component is deprecated"
		}
	}

	for _, t := range doc.all("goimport") {
		fields := strings.Fields(t.text)
		switch len(fields) {
		case 1:
			c.Imports = append(c.Imports, Import{Path: unquote(fields[0])})
		case 2:
			c.Imports = append(c.Imports, Import{Alias: fields[0], Path: unquote(fields[1])})
		default:
			return fmt.Errorf("@goimport takes a path, optionally preceded by an alias")
		}
	}

	for _, t := range doc.all("prop", "property") {
		if err := applyPropDoc(c, t); err != nil {
			return fmt.Errorf("@prop: %w", err)
		}
	}

	for _, t := range doc.all("fires", "event") {
		e, skip, err := parseEventTag(t)
		if err != nil {
			return fmt.Errorf("@fires: %w", err)
		}
		if skip {
			continue
		}
		c.Events = append(c.Events, e)
	}

	for _, t := range doc.all("cssprop", "cssproperty") {
		p, err := parseCSSPropTag(t)
		if err != nil {
			return fmt.Errorf("@cssprop: %w", err)
		}
		c.CSSProps = append(c.CSSProps, p)
	}

	for _, t := range doc.all("slot") {
		c.Slots = append(c.Slots, parseSlotTag(t)...)
	}

	return nil
}

func applyPropDoc(c *Component, t docTag) error {
	typ, rest, hasType := splitType(t.text)
	name, desc := splitPropIdent(rest)
	if name == "" {
		return fmt.Errorf("line %d: no prop name", t.line)
	}

	idx := -1
	for i := range c.Props {
		if c.Props[i].Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		// A documented prop that define() does not declare is a rename that
		// was only half done; generating a field for it would render an
		// attribute the component ignores. It is also the failure when a
		// doc block is attached to the next define() rather than the one
		// it was written for — suggestDocOwner turns that into a better error.
		return unknownPropError{line: t.line, name: name}
	}

	if hasType {
		gt, err := goType(typ)
		if err != nil {
			return fmt.Errorf("line %d: %s", t.line, err)
		}
		c.Props[idx].GoType = gt
	}
	if d := trimDash(desc); d != "" {
		c.Props[idx].Doc = d
	}
	return nil
}

// unknownPropError is a @prop name that define() does not declare. Usually a
// half-done rename; sometimes a doc block that belongs to a later define().
type unknownPropError struct {
	line int
	name string
}

func (e unknownPropError) Error() string {
	return fmt.Sprintf("line %d: %q is not one of the props declared by define()", e.line, e.name)
}

func docPropNames(doc docBlock) []string {
	var names []string
	for _, t := range doc.all("prop", "property") {
		_, rest, _ := splitType(t.text)
		name, _ := splitPropIdent(rest)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func componentHasAllProps(c Component, names []string) bool {
	have := make(map[string]bool, len(c.Props))
	for _, p := range c.Props {
		have[p.Name] = true
	}
	for _, n := range names {
		if !have[n] {
			return false
		}
	}
	return len(names) > 0
}

// suggestDocOwner looks past the define() that just stole a doc block for a
// later component that actually declares those @prop names.
func suggestDocOwner(p *parser, afterDefineIdent int, doc docBlock) (tag string, line int, ok bool) {
	names := docPropNames(doc)
	if len(names) == 0 {
		return "", 0, false
	}
	saved := p.pos
	defer func() { p.pos = saved }()
	for i := afterDefineIdent + 1; i < len(p.toks); i++ {
		t := p.toks[i]
		if t.kind != tIdent || t.val != "define" || !p.at(i+1).is("(") {
			continue
		}
		if p.at(i-1).is(".") && p.at(i-2).kind == tIdent && p.at(i-2).val == "customElements" {
			continue
		}
		p.pos = i + 2
		c, parsed, err := p.parseDefine("")
		if err != nil || !parsed {
			continue
		}
		if componentHasAllProps(c, names) {
			return c.Tag, t.line, true
		}
		if p.pos > i {
			i = p.pos - 1
		}
	}
	return "", 0, false
}

// parseCSSPropTag reads the custom element manifest spelling:
//
//	@cssprop --chip-bg - the background
//	@cssprop [--chip-bg=#eee] - the background, defaulting to #eee
func parseCSSPropTag(t docTag) (CSSProp, error) {
	text := strings.TrimSpace(t.text)

	var p CSSProp
	if inner, ok := strings.CutPrefix(text, "["); ok {
		decl, rest, found := strings.Cut(inner, "]")
		if !found {
			return p, fmt.Errorf("line %d: unclosed [", t.line)
		}
		name, def, _ := strings.Cut(decl, "=")
		p.Name = strings.TrimSpace(name)
		p.Default = strings.TrimSpace(def)
		p.Doc = trimDash(rest)
	} else {
		name, rest, _ := strings.Cut(text, " ")
		p.Name = strings.TrimSpace(name)
		p.Doc = trimDash(rest)
	}

	if !strings.HasPrefix(p.Name, "--") {
		return p, fmt.Errorf("line %d: %q is not a custom property; it needs a -- prefix", t.line, p.Name)
	}
	return p, nil
}

func parseEventTag(t docTag) (Event, bool, error) {
	text := strings.TrimSpace(t.text)

	var (
		e   Event
		typ string
	)
	if typExpr, rest, ok := splitType(text); ok {
		// "{detail} name - description"
		typ, text = typExpr, rest
	}
	name, rest, _ := strings.Cut(text, " ")
	if name == "" {
		return e, false, fmt.Errorf("line %d: no event name", t.line)
	}
	if !isEventName(name) {
		// "@event (native click bubbles; no custom event)" is a note, not
		// an event the generator can name.
		return e, true, nil
	}
	e.Name = name

	if typ == "" {
		// "name {detail} - description"
		if typExpr, after, ok := splitType(rest); ok {
			typ, rest = typExpr, after
		}
	}
	e.Doc = trimDash(rest)

	if typ == "" {
		typ = detailTypeFromDoc(e.Doc)
	}

	switch {
	case typ == "":
	case isObjectType(typ):
		fields, err := parseObjectType(typ)
		if err != nil {
			// Keep the event; a detail we cannot type is still worth a constant.
			break
		}
		e.Detail = fields
	default:
		gt, err := goType(typ)
		if err != nil {
			break
		}
		e.GoType = gt
	}
	return e, false, nil
}

// inferProps fills in each prop's kind, Go type and default from the value in
// define(), leaving anything JSDoc already set alone.
func inferProps(c *Component) error {
	for i := range c.Props {
		p := &c.Props[i]
		raw := c.rawProps[i]

		kind, inferred, goDefault, err := describe(raw.val)
		if err != nil {
			return fmt.Errorf("prop %q: %w", p.Name, err)
		}
		p.Kind = kind
		p.Default = strings.TrimSpace(raw.val.raw)
		p.GoDefault = goDefault
		if p.GoType == "" {
			p.GoType = inferred
		}

		if p.Kind == KindJSON && isScalarGoType(p.GoType) {
			// A scalar Go type would be encoded as a bare attribute value,
			// but the component's default is an object or array, so alacris
			// will run JSON.parse on it and silently fall back to the default.
			return fmt.Errorf("prop %q defaults to %s, so its attribute is parsed as JSON; %s cannot be encoded that way",
				p.Name, raw.val.kind, p.GoType)
		}
	}
	c.rawProps = nil
	return nil
}

func isScalarGoType(t string) bool {
	switch t {
	case "string", "bool", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return true
	}
	return false
}

// describe maps a default value to the coercion alacris will apply, the Go
// type that matches it, and the same default written as a Go expression.
func describe(v jsValue) (kind Kind, goT, goDefault string, err error) {
	switch v.kind {
	case jsString:
		return KindString, "string", strconv.Quote(v.str), nil

	case jsBool:
		return KindBool, "bool", strconv.FormatBool(v.boolean), nil

	case jsNumber:
		if v.integral {
			return KindNumber, "int", strconv.FormatInt(int64(v.num), 10), nil
		}
		return KindNumber, "float64", strconv.FormatFloat(v.num, 'g', -1, 64), nil

	case jsArray:
		if len(v.arr) == 0 {
			return KindJSON, sliceType(v.arr), "", nil
		}
		return KindJSON, sliceType(v.arr), "nonempty", nil

	case jsObject:
		if len(v.obj) == 0 {
			return KindJSON, "map[string]any", "", nil
		}
		return KindJSON, "map[string]any", "nonempty", nil

	case jsNull:
		return KindJSON, "any", "", nil
	}

	return "", "", "", fmt.Errorf(
		"its default is %s, and the type of the default is what decides how the attribute is coerced; "+
			"give it a literal default or declare the type with @prop {type} in the JSDoc above define()", v.kind)
}

// sliceType picks an element type for an array default. An empty array says
// nothing about its elements, so it becomes []any — @prop {string[]} is how to
// say otherwise.
func sliceType(items []jsValue) string {
	if len(items) == 0 {
		return "[]any"
	}
	elem := ""
	for _, it := range items {
		var t string
		switch it.kind {
		case jsString:
			t = "string"
		case jsBool:
			t = "bool"
		case jsNumber:
			if it.integral {
				t = "int"
			} else {
				t = "float64"
			}
		default:
			return "[]any"
		}
		switch {
		case elem == "":
			elem = t
		case elem == t:
		case (elem == "int" && t == "float64") || (elem == "float64" && t == "int"):
			elem = "float64"
		default:
			return "[]any"
		}
	}
	return "[]" + elem
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// SortComponents orders components by tag.
func SortComponents(c []Component) {
	sort.SliceStable(c, func(i, j int) bool { return c[i].Tag < c[j].Tag })
}

// leadingFileDoc reads a documentation block at the start of a source file —
// the // header Alacris UI puts above its imports. ok is false when there
// isn't one, or when it has no @tags (a license banner is not a component doc).
func leadingFileDoc(src string) (docBlock, bool) {
	src = strings.TrimPrefix(src, bom)
	i := 0
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r') {
		i++
	}
	if i >= len(src) {
		return docBlock{}, false
	}
	start := i
	switch {
	case strings.HasPrefix(src[i:], "/**"):
		end := strings.Index(src[i+3:], "*/")
		if end < 0 {
			return docBlock{}, false
		}
		i += 3 + end + 2
	case strings.HasPrefix(src[i:], "//"):
		for i < len(src) {
			for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
				i++
			}
			if !strings.HasPrefix(src[i:], "//") {
				break
			}
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) && src[i] == '\n' {
				i++
			}
		}
	default:
		return docBlock{}, false
	}
	raw := src[start:i]
	if !strings.Contains(raw, "@") {
		return docBlock{}, false
	}
	return parseDoc(raw, 1), true
}

// splitPropIdent pulls a prop name off a @prop tag body, accepting the
// `name='default'` spelling Alacris UI uses in file headers.
func splitPropIdent(rest string) (name, desc string) {
	rest = strings.TrimSpace(rest)
	i := 0
	if i < len(rest) && isIdentStart(rune(rest[i])) {
		i++
		for i < len(rest) && isIdentPart(rune(rest[i])) {
			i++
		}
	}
	if i == 0 {
		return "", rest
	}
	name = rest[:i]
	rest = strings.TrimSpace(rest[i:])
	if strings.HasPrefix(rest, "=") {
		rest = skipJSDefault(strings.TrimSpace(rest[1:]))
	}
	return name, rest
}

// skipJSDefault consumes a default written after `name=` in a @prop tag and
// returns whatever follows it (usually the description).
func skipJSDefault(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '\'', '"', '`':
		q := s[0]
		for i := 1; i < len(s); i++ {
			if s[i] == '\\' {
				i++
				continue
			}
			if s[i] == q {
				return strings.TrimSpace(s[i+1:])
			}
		}
		return ""
	case '[', '{', '(':
		open, close := s[0], byte(map[byte]byte{'[': ']', '{': '}', '(': ')'}[s[0]])
		depth := 0
		for i := 0; i < len(s); i++ {
			switch s[i] {
			case open:
				depth++
			case close:
				depth--
				if depth == 0 {
					return strings.TrimSpace(s[i+1:])
				}
			}
		}
		return ""
	default:
		i := 0
		for i < len(s) {
			if s[i] == ' ' || s[i] == '\t' || strings.HasPrefix(s[i:], "—") || strings.HasPrefix(s[i:], " - ") {
				break
			}
			i++
		}
		return strings.TrimSpace(s[i:])
	}
}

// parseSlotTag reads @slot, including UI spellings: `(default)`, several names
// on one line, and an em-dash before the description.
func parseSlotTag(t docTag) []Slot {
	names, doc := splitTagNames(t.text)
	if len(names) == 0 {
		return []Slot{{Name: "", Doc: doc}}
	}
	out := make([]Slot, 0, len(names))
	for _, n := range names {
		if n == "" || n == "(default)" {
			out = append(out, Slot{Name: "", Doc: doc})
			continue
		}
		out = append(out, Slot{Name: n, Doc: doc})
	}
	return out
}

func splitTagNames(text string) (names []string, doc string) {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(text, "-") {
		return nil, trimDash(text)
	}
	body, doc := text, ""
	if i := strings.Index(text, "—"); i >= 0 {
		body, doc = strings.TrimSpace(text[:i]), trimDash(text[i+len("—"):])
	} else if i := strings.Index(text, " - "); i >= 0 {
		body, doc = strings.TrimSpace(text[:i]), trimDash(text[i+3:])
	}
	if body == "" || body == "(default)" {
		return nil, doc
	}
	for _, part := range strings.Split(body, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		names = append(names, part)
	}
	return names, doc
}

func isEventName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && ((c >= '0' && c <= '9') || c == '-'):
		default:
			return false
		}
	}
	return true
}

// detailTypeFromDoc finds `detail: { ... }` in a free-form event description.
func detailTypeFromDoc(doc string) string {
	lower := strings.ToLower(doc)
	i := strings.Index(lower, "detail:")
	if i < 0 {
		i = strings.Index(lower, "detail ")
		if i < 0 {
			return ""
		}
	}
	rest := strings.TrimSpace(doc[i:])
	if _, after, ok := strings.Cut(strings.ToLower(rest), "detail:"); ok {
		rest = strings.TrimSpace(after)
	} else {
		rest = strings.TrimSpace(rest[len("detail"):])
	}
	typ, _, ok := splitType(rest)
	if !ok {
		return ""
	}
	return "{" + typ + "}"
}

// applyVarsCalls reads vars('prefix', { key: ... }) calls in src and, for any
// component that has no @cssprop tags, fills the contract from the keys.
func applyVarsCalls(src string, cs []Component) {
	byTag := map[string]*Component{}
	for i := range cs {
		byTag[cs[i].Tag] = &cs[i]
	}
	found := varsCalls(src)
	for prefix, keys := range found {
		c, ok := byTag[prefix]
		if !ok {
			if len(cs) == 1 && len(found) == 1 {
				c = &cs[0]
			} else {
				continue
			}
		}
		if len(c.CSSProps) > 0 || len(keys) == 0 {
			continue
		}
		for _, k := range keys {
			c.CSSProps = append(c.CSSProps, CSSProp{
				Name: "--" + prefix + "-" + attrName(k),
			})
		}
	}
}

func varsCalls(src string) map[string][]string {
	p, err := newParser("", src)
	if err != nil {
		return nil
	}
	out := map[string][]string{}
	for i := 0; i < len(p.toks); i++ {
		t := p.toks[i]
		if t.kind != tIdent || t.val != "vars" || !p.at(i+1).is("(") {
			continue
		}
		if p.at(i - 1).is(".") {
			continue
		}
		p.pos = i + 2
		if p.cur().kind != tString {
			continue
		}
		prefix := p.cur().val
		p.advance()
		if !p.cur().is(",") {
			continue
		}
		p.advance()
		if !p.cur().is("{") {
			continue
		}
		obj, err := p.parseObject()
		if err != nil {
			continue
		}
		var keys []string
		for _, m := range obj.obj {
			keys = append(keys, m.key)
		}
		if len(keys) > 0 {
			out[prefix] = keys
		}
	}
	return out
}
