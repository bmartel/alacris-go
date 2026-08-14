package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// DefaultImport is the import path of the runtime package generated code uses.
const DefaultImport = "github.com/bmartel/alacris-go"

// Optional decides how a prop whose default is not the Go zero value is
// represented.
type Optional string

const (
	// OptionalZero uses plain fields. A field left at its zero value, or set
	// to the component's own default, is left off the element. Sending the
	// zero value on purpose means calling Prop on the returned element.
	OptionalZero Optional = "zero"

	// OptionalPointer uses pointer fields, which can say "unset" and "the zero
	// value" separately, at the cost of needing a pointer at every call site.
	OptionalPointer Optional = "pointer"
)

// Options control code generation.
type Options struct {
	// Package is the name of the generated package.
	Package string

	// Import is the import path of the alacris runtime package.
	// Defaults to DefaultImport.
	Import string

	// Optional selects the representation for props with a non-zero default.
	Optional Optional

	// StripPrefix is removed from a tag before deriving its Go name, so a
	// project prefix does not end up on every identifier: with "ala-",
	// <ala-counter> generates Counter.
	StripPrefix string

	// RelativeTo is the directory the source paths in the generated headers
	// are written relative to — normally the output directory.
	//
	// Without it the header records the path exactly as it was typed, so
	// generating from the repository root and from the package directory
	// produce different bytes for the same components, and a check step
	// reports a difference that is not one.
	RelativeTo string

	// Live also generates a typed patch handle per component, so live updates
	// are written with the same compile-checked names as rendering:
	//
	//	ui.TodoListElement(c.Session, "todos").SetItems(items)
	//
	// It makes the generated package import the live package.
	Live bool

	// LiveImport is the import path of the live package. Defaults to
	// Import + "/live".
	LiveImport string
}

// A File is one generated Go source file.
type File struct {
	Name   string
	Source []byte
}

// Generate produces the Go source for a manifest, one file per source file the
// components came from plus a shared file holding the package documentation.
func Generate(m *Manifest, opts Options) ([]File, error) {
	if opts.Package == "" {
		return nil, fmt.Errorf("gen: no package name")
	}
	if opts.Import == "" {
		opts.Import = DefaultImport
	}
	if opts.LiveImport == "" {
		opts.LiveImport = opts.Import + "/live"
	}
	if opts.Optional == "" {
		opts.Optional = OptionalZero
	}
	if opts.Optional != OptionalZero && opts.Optional != OptionalPointer {
		return nil, fmt.Errorf("gen: unknown optional mode %q", opts.Optional)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}

	// Stripping a prefix can make two tags collide where the manifest's own
	// check saw none.
	if opts.StripPrefix != "" {
		seen := map[string]string{}
		for _, c := range m.Components {
			name := identifier(c, opts.StripPrefix)
			if prev, dup := seen[name]; dup {
				return nil, fmt.Errorf("gen: with -strip %q, <%s> and <%s> both generate %s",
					opts.StripPrefix, prev, c.Tag, name)
			}
			seen[name] = c.Tag
		}
	}

	groups := map[string][]Component{}
	var order []string
	for _, c := range m.Components {
		base := "components"
		if c.Source != "" {
			base = fileBase(path.Base(strings.Split(c.Source, ":")[0]))
		}
		if _, ok := groups[base]; !ok {
			order = append(order, base)
		}
		groups[base] = append(groups[base], c)
	}
	sort.Strings(order)

	files := make([]File, 0, len(order)+1)
	for _, base := range order {
		src, err := emitFile(groups[base], opts)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Name: base + "_gen.go", Source: src})
	}

	shared, err := emitShared(m, opts)
	if err != nil {
		return nil, err
	}
	files = append(files, File{Name: "alacris_gen.go", Source: shared})

	return files, nil
}

// relativeSource renders a source path the same way wherever the generator was
// run from. Slashes are forced so a Windows run and a Unix run agree too.
func relativeSource(base, src string) string {
	if base == "" {
		return filepath.ToSlash(src)
	}
	absBase, err1 := filepath.Abs(base)
	absSrc, err2 := filepath.Abs(filepath.FromSlash(src))
	if err1 != nil || err2 != nil {
		return filepath.ToSlash(src)
	}
	rel, err := filepath.Rel(absBase, absSrc)
	if err != nil {
		return filepath.ToSlash(src)
	}
	return filepath.ToSlash(rel)
}

func identifier(c Component, strip string) string {
	if c.GoName != "" {
		return c.GoName
	}
	tag := c.Tag
	if strip != "" {
		if trimmed := strings.TrimPrefix(tag, strip); trimmed != "" {
			tag = trimmed
		}
	}
	return exportedName(tag)
}

type writer struct {
	bytes.Buffer
}

func (w *writer) line(format string, args ...any) {
	if len(args) > 0 {
		fmt.Fprintf(w, format, args...)
	} else {
		w.WriteString(format)
	}
	w.WriteByte('\n')
}

func (w *writer) blank() { w.WriteByte('\n') }

// doc writes a Go doc comment, wrapping the text under the given prefix.
func (w *writer) doc(text string) {
	for _, l := range strings.Split(strings.TrimSpace(text), "\n") {
		l = strings.TrimRight(l, " ")
		if l == "" {
			w.line("//")
			continue
		}
		w.line("// %s", l)
	}
}

func emitShared(m *Manifest, opts Options) ([]byte, error) {
	var w writer
	w.doc(fmt.Sprintf(`Package %s renders the project's alacris components from Go.

Code in this package is generated by alacris-go from the define() calls in the
component sources. Edit the components, or the manifest, not this.

A component's shadow content is rendered in the browser; what these functions
produce is the element, its props and its light-DOM children.`, opts.Package))
	w.line("//")
	w.line("// Code generated by alacris-go. DO NOT EDIT.")
	w.line("package %s", opts.Package)
	w.blank()

	w.doc(`Tags lists every component in this package.

It is what alacris.Pending wants, so that elements do not flash before their
module has loaded:

	@alacris.Pending{Tags: ui.Tags}`)
	w.line("var Tags = []string{")
	tags := m.Tags()
	sort.Strings(tags)
	for _, t := range tags {
		w.line("\t%s,", strconv.Quote(t))
	}
	w.line("}")

	return gofmt(w.Bytes(), "alacris_gen.go")
}

func emitFile(components []Component, opts Options) ([]byte, error) {
	var w writer

	sources := map[string]bool{}
	var sourceList []string
	imports := map[string]Import{}
	for _, c := range components {
		if c.Source != "" {
			src := strings.Split(c.Source, ":")[0]
			if !sources[src] {
				sources[src] = true
				sourceList = append(sourceList, src)
			}
		}
		for _, imp := range c.Imports {
			imports[imp.Path] = imp
		}
	}
	sort.Strings(sourceList)

	if len(sourceList) > 0 {
		shown := make([]string, len(sourceList))
		for i, src := range sourceList {
			shown[i] = relativeSource(opts.RelativeTo, src)
		}
		w.line("// Code generated by alacris-go from %s. DO NOT EDIT.", strings.Join(shown, ", "))
	} else {
		w.line("// Code generated by alacris-go. DO NOT EDIT.")
	}
	w.blank()
	w.line("package %s", opts.Package)
	w.blank()

	w.line("import (")
	w.line("\talacris %s", strconv.Quote(opts.Import))
	if opts.Live {
		w.line("\tlive %s", strconv.Quote(opts.LiveImport))
	}
	paths := make([]string, 0, len(imports))
	for p := range imports {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if alias := imports[p].Alias; alias != "" {
			w.line("\t%s %s", alias, strconv.Quote(p))
			continue
		}
		w.line("\t%s", strconv.Quote(p))
	}
	w.line(")")

	for _, c := range components {
		if err := emitComponent(&w, c, opts); err != nil {
			return nil, err
		}
	}

	name := "components_gen.go"
	if len(sourceList) > 0 {
		name = fileBase(path.Base(sourceList[0])) + "_gen.go"
	}
	return gofmt(w.Bytes(), name)
}

func emitComponent(w *writer, c Component, opts Options) error {
	name := identifier(c, opts.StripPrefix)

	w.blank()
	w.doc(fmt.Sprintf("%sTag is the name of the <%s> custom element.", name, c.Tag))
	w.line("const %sTag = %s", name, strconv.Quote(c.Tag))

	if len(c.Slots) > 0 {
		w.blank()
		w.doc(fmt.Sprintf("Slots exposed by <%s>. Put one on your own markup to fill it:\n\n\t<h3 slot={ %s }>",
			c.Tag, slotIdent(name, c.Slots[0])))
		w.line("const (")
		for _, s := range c.Slots {
			if s.Doc != "" {
				w.line("\t// %s", s.Doc)
			}
			w.line("\t%s = %s", slotIdent(name, s), strconv.Quote(s.Name))
		}
		w.line(")")
	}

	if len(c.Events) > 0 {
		w.blank()
		w.doc(fmt.Sprintf("Events emitted by <%s>. Forward one to a server action with On:\n\n\t%s(props).On(%sEvent%s, \"handler-name\")",
			c.Tag, name, name, exportedName(c.Events[0].Name)))
		w.line("const (")
		for _, e := range c.Events {
			if e.Doc != "" {
				w.line("\t// %s", e.Doc)
			}
			w.line("\t%sEvent%s = %s", name, exportedName(e.Name), strconv.Quote(e.Name))
		}
		w.line(")")

		for _, e := range c.Events {
			if len(e.Detail) == 0 {
				continue
			}
			typeName := name + exportedName(e.Name) + "Detail"
			w.blank()
			doc := fmt.Sprintf("%s is the detail of the %q event of <%s>.", typeName, e.Name, c.Tag)
			if e.Doc != "" {
				doc += "\n\n" + e.Doc
			}
			w.doc(doc)
			w.line("type %s struct {", typeName)
			for _, f := range e.Detail {
				if f.Doc != "" {
					w.line("\t// %s", f.Doc)
				}
				w.line("\t%s %s `json:%s`", exportedName(f.Name), f.GoType, strconv.Quote(f.Name))
			}
			w.line("}")
		}
	}

	if len(c.CSSProps) > 0 {
		w.blank()
		w.doc(fmt.Sprintf(`%sVars is the theming contract of <%s>: the custom properties it
is styled through, and the only supported way to restyle it from outside its
shadow root.

	%s(props).Apply(%sVars, map[string]string{%q: "#111"})

Applying a property that is not in the contract is an error, so a rename in the
component turns up here rather than as styling that silently stops working.`,
			name, c.Tag, name, name, c.CSSProps[0].Name))
		w.line("var %sVars = alacris.Vars(", name)
		for _, p := range c.CSSProps {
			line := "\t" + strconv.Quote(p.Name) + ","
			switch {
			case p.Doc != "" && p.Default != "":
				line += fmt.Sprintf(" // %s (defaults to %s)", p.Doc, p.Default)
			case p.Doc != "":
				line += " // " + p.Doc
			case p.Default != "":
				line += " // defaults to " + p.Default
			}
			w.line("%s", line)
		}
		w.line(")")
	}

	// Props
	w.blank()
	propsDoc := fmt.Sprintf("%sProps are the props of <%s>.", name, c.Tag)
	if opts.Optional == OptionalZero {
		propsDoc += "\n\n" + `A field left at its zero value, or set to the component's own default, is
left off the element, so the component keeps its default. To send a zero
value deliberately, set it with Prop on the returned element.`
	} else {
		propsDoc += "\n\n" + `A nil field is left off the element, so the component keeps its default.`
	}
	w.doc(propsDoc)
	w.line("type %sProps struct {", name)
	for i, p := range c.Props {
		if i > 0 {
			w.blank()
		}
		emitPropDoc(w, p, opts)
		w.line("\t%s %s", p.Field(), fieldType(p, opts))
	}
	w.line("}")

	// Constructor
	w.blank()
	ctorDoc := fmt.Sprintf("%s renders <%s>.", name, c.Tag)
	if c.Doc != "" {
		ctorDoc += "\n\n" + c.Doc
	}
	if c.Deprecated != "" {
		ctorDoc += "\n\nDeprecated: " + c.Deprecated
	}
	w.doc(ctorDoc)
	w.line("func %s(p %sProps) *alacris.Element {", name, name)
	w.line("\te := alacris.E(%sTag)", name)
	for _, p := range c.Props {
		cond, arg := propCondition(p, opts)
		if cond == "" {
			w.line("\te.Prop(%s, %s)", strconv.Quote(p.Name), arg)
			continue
		}
		w.line("\tif %s {", cond)
		w.line("\t\te.Prop(%s, %s)", strconv.Quote(p.Name), arg)
		w.line("\t}")
	}
	w.line("\treturn e")
	w.line("}")

	if opts.Live {
		emitHandle(w, c, name)
	}

	return nil
}

// emitHandle writes the typed live handle for one component: the other half
// of the type-safety story. Rendering already goes through generated names;
// this gives server-driven patches the same treatment, so a renamed prop is a
// compile error rather than a property write nobody reads.
func emitHandle(w *writer, c Component, name string) {
	w.blank()
	w.doc(fmt.Sprintf(`%sHandle patches a rendered <%s> over a live session.

Each setter writes one component prop — one property write, one DOM update on
the page. Obtain one with %sElement.`, name, c.Tag, name))
	w.line("type %sHandle struct {", name)
	w.line("\thandle live.Handle")
	w.line("}")

	w.blank()
	elementDoc := fmt.Sprintf(`%sElement addresses the <%s> rendered with this id, for
patching its props with compile-checked names instead of strings`, name, c.Tag)
	if len(c.Props) > 0 {
		elementDoc += fmt.Sprintf(":\n\n\t%sElement(c.Session, \"id\").Set%s(v)", name, c.Props[0].Field())
	} else {
		elementDoc += "."
	}
	w.doc(elementDoc)
	w.line("func %sElement(s *live.Session, id string) %sHandle {", name, name)
	w.line("\treturn %sHandle{handle: s.Element(id)}", name)
	w.line("}")

	w.blank()
	w.doc(`Handle returns the untyped handle, for attributes, classes and slot HTML.`)
	w.line("func (h %sHandle) Handle() live.Handle { return h.handle }", name)

	for _, p := range c.Props {
		w.blank()
		doc := fmt.Sprintf("Set%s writes the %s prop.", p.Field(), p.Name)
		if p.Deprecated != "" {
			doc += "\n\nDeprecated: " + p.Deprecated
		}
		w.doc(doc)
		w.line("func (h %sHandle) Set%s(v %s) { h.handle.Set(%s, v) }",
			name, p.Field(), p.GoType, strconv.Quote(p.Name))
	}
}

// slotIdent is the constant name for a slot; the default slot has no name of
// its own to build one from.
func slotIdent(component string, s Slot) string {
	if s.Name == "" {
		return component + "SlotDefault"
	}
	return component + "Slot" + exportedName(s.Name)
}

func emitPropDoc(w *writer, p Prop, opts Options) {
	var lines []string
	if p.Doc != "" {
		// JSDoc descriptions are noun phrases; "Field is <phrase>." turns one
		// into the sentence Go doc comments are supposed to start with.
		doc := lowerFirst(p.Doc)
		if !strings.HasSuffix(doc, ".") {
			doc += "."
		}
		lines = append(lines, fmt.Sprintf("%s is %s", p.Field(), doc))
	}
	if p.Default != "" && p.Default != `""` {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, fmt.Sprintf("The component defaults it to %s.", p.Default))
	}
	if p.Kind == KindJSON {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Crosses as JSON in the "+p.Attr+" attribute.")
	}
	if p.Deprecated != "" {
		lines = append(lines, "", "Deprecated: "+p.Deprecated)
	}
	for _, l := range lines {
		if l == "" {
			w.line("\t//")
			continue
		}
		w.line("\t// %s", l)
	}
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	// Only lower an ordinary sentence start, so "URL of the avatar" survives.
	r := []rune(s)
	if len(r) > 1 && r[1] >= 'A' && r[1] <= 'Z' {
		return s
	}
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] += 'a' - 'A'
	}
	return string(r)
}

// fieldType is the Go type of the generated struct field.
func fieldType(p Prop, opts Options) string {
	if pointerProp(p, opts) {
		return "*" + p.GoType
	}
	return p.GoType
}

// pointerProp reports whether a prop needs a pointer field.
//
// A boolean whose default is true always does: false is Go's zero value, so a
// plain field could never mean "leave it alone", and the component's default
// would be unreachable.
func pointerProp(p Prop, opts Options) bool {
	if strings.HasPrefix(p.GoType, "*") {
		return false
	}
	if p.Kind == KindBool && p.GoDefault == "true" {
		return true
	}
	if opts.Optional == OptionalPointer && p.GoDefault != "" && !isZeroLiteral(p.GoDefault) {
		return true
	}
	return false
}

func isZeroLiteral(s string) bool {
	switch s {
	case "", `""`, "0", "false":
		return true
	}
	return false
}

// propCondition returns the guard that decides whether a prop is written, and
// the expression to write. An empty guard means always write it.
func propCondition(p Prop, opts Options) (cond, arg string) {
	field := "p." + p.Field()

	if pointerProp(p, opts) {
		return field + " != nil", "*" + field
	}

	t := p.GoType
	switch {
	case strings.HasPrefix(t, "*"), t == "any", t == "interface{}":
		return field + " != nil", field
	case strings.HasPrefix(t, "[]"), strings.HasPrefix(t, "map["):
		return "len(" + field + ") > 0", field
	}

	if !isScalarGoType(t) {
		// A named or struct type: compare against its zero value reflectively
		// rather than guessing at an operator it may not support.
		return "!alacris.IsZero(" + field + ")", field
	}

	if t == "bool" {
		// The default here is necessarily false; a true default became a
		// pointer above.
		return field, field
	}

	zero := zeroFor(t)
	if p.GoDefault == "" || isZeroLiteral(p.GoDefault) {
		return field + " != " + zero, field
	}
	return field + " != " + zero + " && " + field + " != " + p.GoDefault, field
}

func zeroFor(t string) string {
	switch t {
	case "string":
		return `""`
	case "bool":
		return "false"
	}
	return "0"
}

func gofmt(src []byte, name string) ([]byte, error) {
	out, err := format.Source(src)
	if err != nil {
		return src, fmt.Errorf("gen: %s is not valid Go: %w\n%s", name, err, numberLines(src))
	}
	return out, nil
}

func numberLines(src []byte) string {
	var b strings.Builder
	for i, l := range strings.Split(string(src), "\n") {
		fmt.Fprintf(&b, "%4d | %s\n", i+1, l)
	}
	return b.String()
}
