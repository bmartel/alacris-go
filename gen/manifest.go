package gen

import (
	"encoding/json"
	"fmt"
	goparser "go/parser"
	gotoken "go/token"
	"io"
	"sort"
	"strings"
)

// validGoExpr reports whether s parses as exactly one Go expression — which
// covers type syntax too, since every type the manifest can name is also a
// valid expression production. Trailing tokens, statements and declarations
// all fail, which is the point.
func validGoExpr(s string) bool {
	_, err := goparser.ParseExpr(s)
	return err == nil
}

// ManifestVersion is the schema version written into a manifest file.
const ManifestVersion = 1

// A Manifest is what the generator knows about a set of components.
//
// It is written out deliberately: parsing JavaScript is a best effort, and a
// committed manifest is the escape hatch when a component declares something
// the scanner cannot see — a props object built at runtime, an event whose
// detail is worth a real Go type. Generation reads a manifest and source files
// the same way.
type Manifest struct {
	Version    int         `json:"version"`
	Components []Component `json:"components"`
}

// A Component is one define() call.
type Component struct {
	// Tag is the custom element name.
	Tag string `json:"tag"`

	// GoName overrides the identifier derived from Tag.
	GoName string `json:"goName,omitempty"`

	// Doc is the component's description, from the JSDoc block above define().
	Doc string `json:"doc,omitempty"`

	// Source is where the component was found, as file:line.
	Source string `json:"source,omitempty"`

	// Shadow is "open", "closed" or "none".
	Shadow string `json:"shadow,omitempty"`

	// Deprecated carries the reason, when the component is marked as such.
	Deprecated string `json:"deprecated,omitempty"`

	Props    []Prop    `json:"props,omitempty"`
	Events   []Event   `json:"events,omitempty"`
	Slots    []Slot    `json:"slots,omitempty"`
	CSSProps []CSSProp `json:"cssProps,omitempty"`
	Imports  []Import  `json:"imports,omitempty"`

	// rawProps holds the parsed defaults between reading define() and working
	// out each prop's type. It never reaches the manifest file.
	rawProps []rawProp
}

// Kind is how alacris coerces an attribute for a prop, decided by the type of
// the prop's default in define().
type Kind string

const (
	KindString Kind = "string"
	KindNumber Kind = "number"
	KindBool   Kind = "boolean"
	KindJSON   Kind = "json"
)

// A Prop is one entry of a component's props object.
type Prop struct {
	// Name is the JavaScript prop name, as written in define().
	Name string `json:"name"`

	// Attr is the attribute alacris observes for it.
	Attr string `json:"attr"`

	// Kind is the coercion alacris applies.
	Kind Kind `json:"kind"`

	// GoType is the type of the generated struct field.
	GoType string `json:"goType"`

	// GoName overrides the field name derived from Name.
	GoName string `json:"goName,omitempty"`

	// Default is the prop's default, as JavaScript source. A prop set to its
	// default is left off the element.
	Default string `json:"default,omitempty"`

	// GoDefault is Default as a Go expression, when one can be written.
	GoDefault string `json:"goDefault,omitempty"`

	Doc        string `json:"doc,omitempty"`
	Deprecated string `json:"deprecated,omitempty"`
}

// An Event is a CustomEvent the component emits, declared with @fires.
type Event struct {
	Name string `json:"name"`
	Doc  string `json:"doc,omitempty"`

	// GoType names an existing Go type for the detail instead of generating
	// one from Detail.
	GoType string `json:"goType,omitempty"`

	// Detail describes the fields of the event's detail object.
	Detail []Field `json:"detail,omitempty"`
}

// A Field is one member of an event detail object.
type Field struct {
	Name   string `json:"name"`
	GoType string `json:"goType"`
	Doc    string `json:"doc,omitempty"`
}

// A Slot is a named slot the component renders, declared with @slot.
type Slot struct {
	// Name is empty for the default slot.
	Name string `json:"name"`
	Doc  string `json:"doc,omitempty"`
}

// A CSSProp is a custom property the component is themed by — one entry of its
// vars() contract, declared with @cssprop.
//
// It is the part of a component a consumer is meant to reach through the
// shadow boundary, so it is worth having in Go: a renamed property becomes a
// compile-time-adjacent error rather than styling that quietly stops working.
type CSSProp struct {
	// Name is the full property name, including the leading dashes.
	Name string `json:"name"`

	// Default is the fallback baked into the component's own stylesheet.
	Default string `json:"default,omitempty"`

	Doc string `json:"doc,omitempty"`
}

// An Import is a package the generated file needs for a hand-declared Go type.
type Import struct {
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
}

// Sort orders a manifest so generated output does not depend on the order the
// files happened to be read in.
func (m *Manifest) Sort() {
	sort.SliceStable(m.Components, func(i, j int) bool {
		return m.Components[i].Tag < m.Components[j].Tag
	})
}

// Tags lists every component tag.
func (m *Manifest) Tags() []string {
	out := make([]string, len(m.Components))
	for i, c := range m.Components {
		out[i] = c.Tag
	}
	return out
}

// Validate reports problems that would produce Go code that does not compile,
// or Go code that quietly disagrees with the component.
func (m *Manifest) Validate() error {
	var errs []string
	seen := map[string]string{}
	goNames := map[string]string{}

	for i := range m.Components {
		c := &m.Components[i]
		where := c.Tag
		if c.Source != "" {
			where = c.Source
		}
		if c.Tag == "" {
			errs = append(errs, fmt.Sprintf("%s: component has no tag", where))
			continue
		}
		if !strings.Contains(c.Tag, "-") {
			errs = append(errs, fmt.Sprintf("%s: %q is not a custom element name (it needs a hyphen)", where, c.Tag))
		}
		if prev, dup := seen[c.Tag]; dup {
			errs = append(errs, fmt.Sprintf("%s: <%s> is also defined at %s", where, c.Tag, prev))
		}
		seen[c.Tag] = where

		name := c.Identifier()
		if prev, dup := goNames[name]; dup {
			errs = append(errs, fmt.Sprintf("%s: <%s> and <%s> both generate %s; set goName on one of them",
				where, c.Tag, prev, name))
		}
		goNames[name] = c.Tag

		// The goName, goType and goDefault strings are written into the
		// generated file verbatim, so anything that is not the single
		// identifier or expression it claims to be is refused here rather
		// than smuggled into Go source. gofmt would catch invalid Go; this
		// catches valid Go that is more than what the field means.
		if c.GoName != "" && !gotoken.IsIdentifier(c.GoName) {
			errs = append(errs, fmt.Sprintf("%s: goName %q is not a Go identifier", where, c.GoName))
		}

		fields := map[string]string{}
		for _, p := range c.Props {
			if p.Name == "" {
				errs = append(errs, fmt.Sprintf("%s: <%s> has a prop with no name", where, c.Tag))
				continue
			}
			if p.GoType == "" {
				errs = append(errs, fmt.Sprintf("%s: <%s>.%s has no Go type", where, c.Tag, p.Name))
			} else if !validGoExpr(p.GoType) {
				errs = append(errs, fmt.Sprintf("%s: <%s>.%s: goType %q is not a type", where, c.Tag, p.Name, p.GoType))
			}
			if p.GoName != "" && !gotoken.IsIdentifier(p.GoName) {
				errs = append(errs, fmt.Sprintf("%s: <%s>.%s: goName %q is not a Go identifier", where, c.Tag, p.Name, p.GoName))
			}
			if p.GoDefault != "" && !validGoExpr(p.GoDefault) {
				errs = append(errs, fmt.Sprintf("%s: <%s>.%s: goDefault %q is not an expression", where, c.Tag, p.Name, p.GoDefault))
			}
			f := p.Field()
			if prev, dup := fields[f]; dup {
				errs = append(errs, fmt.Sprintf("%s: <%s> props %q and %q both generate field %s",
					where, c.Tag, prev, p.Name, f))
			}
			fields[f] = p.Name
		}

		events := map[string]bool{}
		for _, e := range c.Events {
			if e.Name == "" {
				errs = append(errs, fmt.Sprintf("%s: <%s> has an event with no name", where, c.Tag))
			}
			if events[e.Name] {
				errs = append(errs, fmt.Sprintf("%s: <%s> declares @fires %s twice", where, c.Tag, e.Name))
			}
			events[e.Name] = true
			if e.GoType != "" && !validGoExpr(e.GoType) {
				errs = append(errs, fmt.Sprintf("%s: <%s> @fires %s: goType %q is not a type", where, c.Tag, e.Name, e.GoType))
			}
			for _, f := range e.Detail {
				if f.GoType != "" && !validGoExpr(f.GoType) {
					errs = append(errs, fmt.Sprintf("%s: <%s> @fires %s.%s: goType %q is not a type", where, c.Tag, e.Name, f.Name, f.GoType))
				}
			}
		}

		for _, imp := range c.Imports {
			if imp.Alias != "" && !gotoken.IsIdentifier(imp.Alias) {
				errs = append(errs, fmt.Sprintf("%s: <%s>: import alias %q is not a Go identifier", where, c.Tag, imp.Alias))
			}
			if strings.ContainsAny(imp.Path, "\"`\n\r") {
				errs = append(errs, fmt.Sprintf("%s: <%s>: import path %q is not an import path", where, c.Tag, imp.Path))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid manifest:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// Identifier is the Go name for the component.
func (c Component) Identifier() string {
	if c.GoName != "" {
		return c.GoName
	}
	return exportedName(c.Tag)
}

// Field is the Go struct field name for the prop.
func (p Prop) Field() string {
	if p.GoName != "" {
		return p.GoName
	}
	return exportedName(p.Name)
}

// WriteManifest writes m as indented JSON.
func WriteManifest(w io.Writer, m *Manifest) error {
	m.Version = ManifestVersion
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

// ReadManifest parses a manifest, filling in anything derivable that a
// hand-edited file left out.
func ReadManifest(r io.Reader) (*Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	if m.Version > ManifestVersion {
		return nil, fmt.Errorf("manifest version %d is newer than this tool understands (%d)", m.Version, ManifestVersion)
	}
	for i := range m.Components {
		c := &m.Components[i]
		for j := range c.Props {
			p := &c.Props[j]
			if p.Attr == "" {
				p.Attr = attrName(p.Name)
			}
			if p.Kind == "" {
				p.Kind = KindJSON
			}
			if p.GoType == "" {
				p.GoType = goTypeForKind(p.Kind)
			}
		}
	}
	return &m, nil
}

func goTypeForKind(k Kind) string {
	switch k {
	case KindString:
		return "string"
	case KindNumber:
		return "float64"
	case KindBool:
		return "bool"
	}
	return "any"
}
