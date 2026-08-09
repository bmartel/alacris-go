// Command docsgen renders the documentation site's Go examples.
//
// Each example in examples.go is extracted from the source with go/ast and
// then executed, so the code on the page and the HTML beside it come from one
// place and cannot disagree. The result is written to
// docs/src/generated/examples.json, which the site imports.
//
//	go run ./internal/docsgen            # write the file
//	go run ./internal/docsgen -check     # fail if it is out of date
//
// The -check form runs in CI and in TestDocExamplesAreCurrent, so an example
// that stopped matching the library is a failing build rather than a page that
// quietly documents something that no longer happens.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"

	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
)

// An example is one entry on the site.
type example struct {
	id   string
	fn   string
	tags []string
	note string
}

// Rendered is the JSON shape the site consumes.
type Rendered struct {
	ID string `json:"id"`

	// Go is the example's source, exactly as written in examples.go.
	Go string `json:"go"`

	// HTML is what that source renders.
	HTML string `json:"html"`

	// Pretty is HTML broken across lines so a long element is readable.
	Pretty string `json:"pretty"`

	// Tags are the custom elements the output contains, so a page can load
	// stand-ins for them and show the result running.
	Tags []string `json:"tags,omitempty"`

	// Note is the one-line point the example is making.
	Note string `json:"note,omitempty"`
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("docsgen: ")

	check := flag.Bool("check", false, "verify the committed file is current, and write nothing")
	out := flag.String("o", filepath.Join("docs", "src", "generated", "examples.json"), "file to write")
	src := flag.String("src", filepath.Join("internal", "docsgen", "examples.go"), "the source examples are read from")
	flag.Parse()

	status, err := generate(*src, *out, *check)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(status)
}

// generate renders every example and writes the site's JSON. With check set it
// only reports whether the committed file is current.
func generate(src, out string, check bool) (string, error) {
	bodies, err := extractBodies(src)
	if err != nil {
		return "", err
	}

	rendered := make([]Rendered, 0, len(catalogue))
	for _, ex := range catalogue {
		body, ok := bodies[ex.fn]
		if !ok {
			return "", fmt.Errorf("example %q names %s, which is not in %s", ex.id, ex.fn, src)
		}

		component := render(ex.fn)
		if component == nil {
			return "", fmt.Errorf("example %q is in the catalogue but not wired into render()", ex.id)
		}

		var buf bytes.Buffer
		ctx := templ.InitializeContext(context.Background())
		if err := component.Render(ctx, &buf); err != nil {
			return "", fmt.Errorf("example %q: %w", ex.id, err)
		}

		rendered = append(rendered, Rendered{
			ID:     ex.id,
			Go:     body,
			HTML:   buf.String(),
			Pretty: prettyHTML(buf.String()),
			Tags:   ex.tags,
			Note:   ex.note,
		})
	}

	body, err := json.MarshalIndent(rendered, "", "  ")
	if err != nil {
		return "", err
	}
	body = append(body, '\n')

	existing, readErr := os.ReadFile(out)
	current := readErr == nil && bytes.Equal(existing, body)

	if current {
		return fmt.Sprintf("%d example(s) up to date", len(rendered)), nil
	}
	if check {
		return "", fmt.Errorf("%s is out of date\n\trun: go run ./internal/docsgen", out)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(out, body, 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %d example(s) to %s", len(rendered), out), nil
}

// extractBodies reads the statements of every example function, so the source
// on the page is the source that ran rather than a copy of it.
func extractBodies(path string) (map[string]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	out := map[string]string{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "example") {
			continue
		}

		var b strings.Builder
		for i, stmt := range fn.Body.List {
			if i > 0 {
				b.WriteString("\n")
			}
			// Comments inside the body are part of the explanation, so they
			// are taken from the raw source rather than reprinted from the AST.
			start := fset.Position(stmt.Pos()).Offset
			end := fset.Position(stmt.End()).Offset
			b.WriteString(withLeadingComments(string(src), fn, fset, stmt, start, end))
		}
		out[fn.Name.Name] = dedent(b.String())
	}
	return out, nil
}

// withLeadingComments returns a statement's source together with any comment
// lines sitting directly above it.
func withLeadingComments(src string, fn *ast.FuncDecl, fset *token.FileSet, stmt ast.Stmt, start, end int) string {
	lineStart := strings.LastIndexByte(src[:start], '\n') + 1
	bodyStart := fset.Position(fn.Body.Lbrace).Offset

	// Walk backwards over whole lines while they are comments.
	for lineStart > bodyStart {
		prevEnd := lineStart - 1
		prevStart := strings.LastIndexByte(src[:prevEnd], '\n') + 1
		if prevStart < bodyStart {
			break
		}
		if !strings.HasPrefix(strings.TrimSpace(src[prevStart:prevEnd]), "//") {
			break
		}
		lineStart = prevStart
	}
	return src[lineStart:end]
}

// dedent removes the common leading tabs a function body carries.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	depth := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, "\t"))
		if depth < 0 || n < depth {
			depth = n
		}
	}
	if depth <= 0 {
		return s
	}
	for i, l := range lines {
		if len(l) >= depth {
			lines[i] = l[depth:]
		}
	}
	return strings.Join(lines, "\n")
}

// wrapWidth is where a start tag stops fitting on one line.
const wrapWidth = 72

// prettyHTML reformats rendered output for reading.
//
// What the library emits is deliberately compact, which is right for the wire
// and unreadable on a page: a single line carrying a JSON prop with escaped
// quotes tells a reader nothing. This indents the tree and, for a start tag
// that does not fit, puts each attribute on its own line.
func prettyHTML(s string) string {
	nodes, _ := parseNodes(s, 0)
	var b strings.Builder
	writeNodes(&b, nodes, 0)
	return strings.TrimRight(b.String(), "\n")
}

type node struct {
	// tag is empty for a text node, in which case text holds the content.
	tag      string
	attrs    []string
	text     string
	children []*node
	void     bool
}

// parseNodes reads siblings until an end tag or the end of the input, and
// returns where it stopped. The input is this library's own output, so the
// scanner only has to handle what the library can produce.
func parseNodes(s string, i int) ([]*node, int) {
	var out []*node
	for i < len(s) {
		if s[i] != '<' {
			start := i
			for i < len(s) && s[i] != '<' {
				i++
			}
			if t := s[start:i]; strings.TrimSpace(t) != "" {
				out = append(out, &node{text: t})
			}
			continue
		}
		if strings.HasPrefix(s[i:], "</") {
			// An end tag belongs to the caller.
			close := strings.IndexByte(s[i:], '>')
			if close < 0 {
				return out, len(s)
			}
			return out, i + close + 1
		}

		n, next := parseTag(s, i)
		i = next
		if !n.void {
			n.children, i = parseNodes(s, i)
		}
		out = append(out, n)
	}
	return out, i
}

// voidElements never have children or an end tag.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"source": true, "track": true, "wbr": true,
}

func parseTag(s string, i int) (*node, int) {
	i++ // past '<'
	start := i
	for i < len(s) && s[i] != ' ' && s[i] != '>' && s[i] != '/' {
		i++
	}
	n := &node{tag: s[start:i]}

	for i < len(s) && s[i] != '>' {
		for i < len(s) && s[i] == ' ' {
			i++
		}
		if i >= len(s) || s[i] == '>' || s[i] == '/' {
			break
		}
		attrStart := i
		inQuotes := false
		for i < len(s) {
			c := s[i]
			if c == '"' {
				inQuotes = !inQuotes
			} else if !inQuotes && (c == ' ' || c == '>') {
				break
			}
			i++
		}
		n.attrs = append(n.attrs, s[attrStart:i])
	}
	for i < len(s) && s[i] != '>' {
		i++
	}
	n.void = voidElements[n.tag]
	return n, i + 1 // past '>'
}

func writeNodes(b *strings.Builder, nodes []*node, depth int) {
	pad := strings.Repeat("  ", depth)
	for _, n := range nodes {
		if n.tag == "" {
			b.WriteString(pad + strings.TrimSpace(n.text) + "\n")
			continue
		}

		open := "<" + n.tag
		if len(n.attrs) > 0 {
			open += " " + strings.Join(n.attrs, " ")
		}
		open += ">"

		// Break the attributes out only when the tag does not fit.
		if len(pad)+len(open) > wrapWidth && len(n.attrs) > 0 {
			b.WriteString(pad + "<" + n.tag + "\n")
			for _, a := range n.attrs {
				b.WriteString(pad + "  " + a + "\n")
			}
			b.WriteString(pad + ">")
		} else {
			b.WriteString(pad + open)
		}

		switch {
		case n.void:
			b.WriteString("\n")

		case len(n.children) == 0:
			b.WriteString("</" + n.tag + ">\n")

		// A lone short text child stays on the same line, which is how anyone
		// would write it by hand.
		case len(n.children) == 1 && n.children[0].tag == "" &&
			len(strings.TrimSpace(n.children[0].text)) < 40:
			b.WriteString(strings.TrimSpace(n.children[0].text) + "</" + n.tag + ">\n")

		default:
			b.WriteString("\n")
			writeNodes(b, n.children, depth+1)
			b.WriteString(pad + "</" + n.tag + ">\n")
		}
	}
}
