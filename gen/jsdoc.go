package gen

import (
	"fmt"
	"strings"
)

// A component's props object says what a prop is called and how it is coerced,
// but not what an event carries, which slots exist, or that a number is really
// an integer. JSDoc above the define() call fills that in:
//
//	/**
//	 * A person, at a glance.
//	 *
//	 * @prop {string[]} tags  the labels shown under the name
//	 * @fires greet {name: string} - the user said hello
//	 * @slot  title - replaces the heading
//	 * @goname PersonCard
//	 */
//	define('user-card', { props: { name: 'anon', tags: [] }, setup });
//
// Only tags that change the generated Go are read; anything else is left for
// whatever other tooling wants it.

type docBlock struct {
	description string
	tags        []docTag
}

type docTag struct {
	name string // without the '@'
	text string
	line int
}

// parseDoc strips comment markers and splits a doc comment into its
// description and its tags. line is the source line the comment starts on.
func parseDoc(raw string, line int) docBlock {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "/**")
	raw = strings.TrimSuffix(raw, "*/")

	var (
		block   docBlock
		descr   []string
		current *docTag
	)

	for i, l := range strings.Split(raw, "\n") {
		text := strings.TrimSpace(l)
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimSpace(text)
		text = strings.TrimPrefix(text, "*")
		text = strings.TrimSpace(text)

		if strings.HasPrefix(text, "@") {
			if current != nil {
				block.tags = append(block.tags, *current)
			}
			name, rest, _ := strings.Cut(text[1:], " ")
			current = &docTag{name: strings.ToLower(name), text: strings.TrimSpace(rest), line: line + i}
			continue
		}

		if current != nil {
			// A wrapped tag continues on the next line.
			if text == "" {
				block.tags = append(block.tags, *current)
				current = nil
				continue
			}
			current.text = strings.TrimSpace(current.text + " " + text)
			continue
		}
		descr = append(descr, text)
	}
	if current != nil {
		block.tags = append(block.tags, *current)
	}

	block.description = strings.TrimSpace(strings.Join(descr, "\n"))
	return block
}

func (d docBlock) find(name string) (docTag, bool) {
	for _, t := range d.tags {
		if t.name == name {
			return t, true
		}
	}
	return docTag{}, false
}

func (d docBlock) all(names ...string) []docTag {
	var out []docTag
	for _, t := range d.tags {
		for _, n := range names {
			if t.name == n {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

// splitType pulls a leading {…} type expression off a tag body, keeping braces
// balanced so an inline object type survives.
func splitType(s string) (typ, rest string, ok bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") {
		return "", s, false
	}
	depth := 0
	for i, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[1:i]), strings.TrimSpace(s[i+1:]), true
			}
		}
	}
	return "", s, false
}

// trimDash removes the "- " that JSDoc conventionally puts before a
// description.
func trimDash(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "—")
	return strings.TrimSpace(s)
}

// goType converts a JSDoc type expression to a Go type.
//
// The vocabulary is deliberately small. "number" becomes float64 because that
// is what a JavaScript number is; write "integer" when the value really is one
// and the generated code should say int. "go:T" passes T through untouched,
// for a type declared by hand in the target package.
func goType(expr string) (string, error) {
	e := strings.TrimSpace(expr)
	if e == "" {
		return "", fmt.Errorf("empty type")
	}

	if rest, ok := strings.CutPrefix(e, "go:"); ok {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return "", fmt.Errorf("go: needs a type after it")
		}
		return rest, nil
	}

	// Nullable and optional markers carry no extra information here.
	e = strings.TrimPrefix(e, "?")
	e = strings.TrimPrefix(e, "!")
	e = strings.TrimSuffix(e, "=")
	e = strings.TrimSpace(e)

	if strings.Contains(e, "|") {
		// A union (`string | number`) has no single Go type. any lets the
		// caller pass whichever alternative they mean; EncodeProp still
		// types the value they actually send.
		return "any", nil
	}

	if inner, ok := strings.CutSuffix(e, "[]"); ok {
		elem, err := goType(inner)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil
	}
	if inner, ok := cutGeneric(e, "Array"); ok {
		elem, err := goType(inner)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil
	}
	if inner, ok := cutGeneric(e, "Record"); ok {
		// Record<string, T>: only the value type reaches Go.
		if _, val, found := strings.Cut(inner, ","); found {
			elem, err := goType(val)
			if err != nil {
				return "", err
			}
			return "map[string]" + elem, nil
		}
	}

	switch strings.ToLower(e) {
	case "string":
		return "string", nil
	case "number", "float":
		return "float64", nil
	case "integer", "int":
		return "int", nil
	case "boolean", "bool":
		return "bool", nil
	case "object":
		return "map[string]any", nil
	case "array":
		return "[]any", nil
	case "any", "*", "unknown", "json":
		return "any", nil
	}

	return "", fmt.Errorf("unknown type %q (use string, number, integer, boolean, object, any, T[], or go:YourType)", expr)
}

func cutGeneric(e, name string) (string, bool) {
	if !strings.HasPrefix(e, name+"<") || !strings.HasSuffix(e, ">") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(e, name+"<"), ">"), true
}

// isObjectType reports whether an expression is an inline object type such as
// "{name: string, count: number}" or the unbraced "name: string" form
// splitType returns, rather than a named type.
func isObjectType(expr string) bool {
	e := strings.TrimSpace(expr)
	if strings.HasPrefix(e, "go:") {
		return false
	}
	if strings.HasPrefix(e, "{") && strings.HasSuffix(e, "}") {
		return true
	}
	return strings.Contains(e, ":")
}

// parseObjectType reads an inline object type into fields.
func parseObjectType(expr string) ([]Field, error) {
	e := strings.TrimSpace(expr)
	e = strings.TrimPrefix(e, "{")
	e = strings.TrimSuffix(e, "}")

	var out []Field
	for _, part := range splitTopLevel(e, ',') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, typ, found := strings.Cut(part, ":")
		name = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(name), "?"))
		if name == "" {
			continue
		}
		if !found {
			// "{ value }" — a named field whose type was not given.
			out = append(out, Field{Name: name, GoType: "any"})
			continue
		}
		gt, err := goType(typ)
		if err != nil {
			// A union, a quoted literal, anything we do not have a Go type
			// for: keep the field and say any rather than drop the event.
			out = append(out, Field{Name: name, GoType: "any"})
			continue
		}
		out = append(out, Field{Name: name, GoType: gt})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no fields")
	}
	return out, nil
}

// splitTopLevel splits on sep, ignoring separators nested in brackets or
// angle brackets.
func splitTopLevel(s string, sep rune) []string {
	var (
		out   []string
		depth int
		start int
	)
	for i, r := range s {
		switch r {
		case '{', '[', '(', '<':
			depth++
		case '}', ']', ')', '>':
			depth--
		case sep:
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + len(string(sep))
			}
		}
	}
	return append(out, s[start:])
}
