package alacris

import (
	"context"
	"io"
	"sort"
	"strings"

	"github.com/a-h/templ"
)

// VarSet is the Go side of a component's theming contract — the custom
// properties declared by alacris' vars(prefix, defaults).
//
// Names are derived exactly as style.js derives them, so a Go constant and the
// component's own stylesheet cannot drift apart:
//
//	vars('btn', { bg: '#111' })  ->  --btn-bg
type VarSet struct {
	names []string          // full property names, sorted
	byKey map[string]string // every accepted spelling -> full name
}

// NewVarSet declares a contract the way alacris' vars() does: a prefix plus
// camelCase keys. NewVarSet("btn", "bg", "borderRadius") is the contract for
// vars('btn', { bg: ..., borderRadius: ... }), giving --btn-bg and
// --btn-border-radius.
//
// Both the short key and the full property name work as keys in Apply.
func NewVarSet(prefix string, keys ...string) VarSet {
	v := VarSet{byKey: map[string]string{}}
	for _, key := range keys {
		full := "--" + prefix + "-" + AttrName(key)
		v.names = append(v.names, full)
		v.byKey[key] = full
		v.byKey[full] = full
	}
	sort.Strings(v.names)
	return v
}

// Vars declares a contract from full custom property names, which is what
// generated packages use: a component's @cssprop tags name the properties
// outright rather than a prefix and keys.
func Vars(names ...string) VarSet {
	v := VarSet{byKey: map[string]string{}}
	for _, name := range names {
		full := name
		if !strings.HasPrefix(full, "--") {
			full = "--" + full
		}
		v.names = append(v.names, full)
		v.byKey[full] = full
		v.byKey[strings.TrimPrefix(full, "--")] = full
	}
	sort.Strings(v.names)
	return v
}

// Name returns the full custom property name for a key, or "" when the key is
// not part of the contract.
func (v VarSet) Name(key string) string { return v.byKey[key] }

// Names lists every declared property, which is the documented surface a
// consumer can override.
func (v VarSet) Names() []string {
	return append([]string(nil), v.names...)
}

// Apply sets the given values on an element. A key outside the contract is
// recorded as an error on the element rather than written out, so a rename in
// the component surfaces here instead of silently theming nothing.
func (v VarSet) Apply(e *Element, values map[string]string) *Element {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		full, ok := v.byKey[k]
		if !ok {
			e.fail(&unknownVarError{key: k, known: v.names})
			continue
		}
		e.Style(full, values[k])
	}
	return e
}

type unknownVarError struct {
	key   string
	known []string
}

func (e *unknownVarError) Error() string {
	return "alacris: " + e.key + " is not part of this theming contract (have: " +
		strings.Join(e.known, ", ") + ")"
}

// Pending renders the stylesheet that hides custom elements until they are
// defined.
//
// It is worth having. An alacris component's shadow content is built by
// setup() in the browser, so between first paint and the module loading, the
// element is present but empty. Without this the page paints, then reflows.
//
// The elements still occupy no space until they are defined; give them a
// reserved size in your own CSS if layout stability matters.
type Pending struct {
	// Tags to hide. Usually every alacris tag the page renders.
	Tags []string

	// Style is the declaration applied while undefined.
	// Defaults to visibility: hidden.
	Style map[string]string

	// Nonce for the emitted <style> element. Falls back to the context nonce.
	Nonce string
}

var _ templ.Component = Pending{}

// Render writes the stylesheet.
func (p Pending) Render(ctx context.Context, w io.Writer) error {
	if len(p.Tags) == 0 {
		return nil
	}
	selectors := make([]string, 0, len(p.Tags))
	for _, tag := range p.Tags {
		if err := ValidTagName(tag); err != nil {
			return err
		}
		selectors = append(selectors, tag+":not(:defined)")
	}

	decls := p.Style
	if len(decls) == 0 {
		decls = map[string]string{"visibility": "hidden"}
	}
	keys := make([]string, 0, len(decls))
	for k := range decls {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	nonce := p.Nonce
	if nonce == "" {
		nonce = templ.GetNonce(ctx)
	}
	b.WriteString("<style")
	if nonce != "" {
		b.WriteString(` nonce="` + templ.EscapeString(nonce) + `"`)
	}
	b.WriteByte('>')
	b.WriteString(strings.Join(selectors, ","))
	b.WriteByte('{')
	for i, k := range keys {
		prop, val, err := sanitizeDeclaration(k, decls[k])
		if err != nil {
			return err
		}
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(prop)
		b.WriteByte(':')
		b.WriteString(val)
	}
	b.WriteString("}</style>")

	_, err := io.WriteString(w, b.String())
	return err
}
