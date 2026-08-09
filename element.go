package alacris

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/a-h/templ"
)

// Element is a server-rendered alacris custom element: a tag, its props, its
// ordinary HTML attributes, and its light-DOM children.
//
// It implements templ.Component, so it can be used directly in a .templ file:
//
//	@alacris.E("user-card").Prop("name", "Ada").Prop("tags", []string{"a", "b"}) {
//	    <h3 slot="title">Ada Lovelace</h3>
//	}
//
// What the server can render is the tag, its attributes and its children.
// A component's shadow content is built by setup() in the browser and never
// appears in the HTML — see the package documentation.
//
// Builder methods collect errors instead of panicking; Render reports them.
type Element struct {
	tag      string
	props    []kv
	attrs    []kv
	classes  []string
	styles   []kv
	events   []string
	slots    []templ.Component
	children templ.Component
	errs     []error
}

type kv struct {
	key   string
	value string
	bare  bool
}

var _ templ.Component = (*Element)(nil)

// E starts an element with the given custom element tag name. An invalid tag
// name is recorded as an error and surfaces when the element is rendered.
func E(tag string) *Element {
	e := &Element{tag: tag}
	if err := ValidTagName(tag); err != nil {
		e.fail(err)
	}
	return e
}

func (e *Element) fail(err error) {
	e.errs = append(e.errs, err)
}

// Tag returns the element's tag name.
func (e *Element) Tag() string { return e.tag }

// Err reports the first problem recorded while building the element, if any.
func (e *Element) Err() error { return errors.Join(e.errs...) }

// Prop sets a component prop, converting name to the attribute alacris
// observes for it and encoding the value for that prop's declared type. A nil
// value leaves the attribute off so the component keeps its own default.
//
// Prop values are encoded from their Go type (see EncodeProp), so the Go type
// has to agree with the type of the prop's default in define(). Generated
// wrappers guarantee that; hand-written calls are on their own.
func (e *Element) Prop(name string, v any) *Element {
	s, ok, err := EncodeProp(v)
	if err != nil {
		e.fail(fmt.Errorf("prop %q on <%s>: %w", name, e.tag, err))
		return e
	}
	if !ok {
		return e
	}
	return e.setProp(name, s)
}

// setProp converts a prop name to its attribute and checks it before it goes
// anywhere near the start tag.
//
// The name is not escapable — it is written verbatim, because escaping it
// would change which attribute it is. Generated wrappers pass literals, but
// Props takes a map, and a map is the sort of thing that ends up holding data.
func (e *Element) setProp(name, value string) *Element {
	attr := AttrName(name)
	if err := ValidAttrName(attr); err != nil {
		e.fail(fmt.Errorf("prop %q on <%s>: %w", name, e.tag, err))
		return e
	}
	return e.set(&e.props, attr, value)
}

// PropRaw sets a prop to an already-encoded attribute value, skipping the
// encoding rules. The caller is responsible for matching the prop's declared
// type — a prop whose default is an object needs valid JSON here, and alacris
// silently falls back to the default when JSON.parse throws.
func (e *Element) PropRaw(name, value string) *Element {
	return e.setProp(name, value)
}

// Props sets several props at once. Keys are sorted so output is stable.
func (e *Element) Props(props map[string]any) *Element {
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		e.Prop(name, props[name])
	}
	return e
}

// Attr sets an ordinary HTML attribute, using HTML's rules rather than
// alacris' — a true bool renders bare and a false one is omitted.
//
// "class" appends to the element's class list. "style" is rejected in favour
// of Style and Var, which sanitize their input.
func (e *Element) Attr(name string, v any) *Element {
	switch name {
	case "class":
		s, _, ok, err := EncodeAttr(v)
		if err != nil {
			e.fail(fmt.Errorf("attribute %q on <%s>: %w", name, e.tag, err))
			return e
		}
		if ok {
			e.Class(strings.Fields(s)...)
		}
		return e
	case "style":
		e.fail(fmt.Errorf("attribute \"style\" on <%s>: use Style or Var so the declaration is sanitized", e.tag))
		return e
	}
	if err := ValidAttrName(name); err != nil {
		e.fail(fmt.Errorf("on <%s>: %w", e.tag, err))
		return e
	}
	s, bare, ok, err := EncodeAttr(v)
	if err != nil {
		e.fail(fmt.Errorf("attribute %q on <%s>: %w", name, e.tag, err))
		return e
	}
	if !ok {
		return e
	}
	if bare {
		return e.setBare(&e.attrs, name)
	}
	return e.set(&e.attrs, name, s)
}

// Attrs spreads a templ attribute map onto the element. Keys are applied in
// sorted order.
func (e *Element) Attrs(attrs templ.Attributes) *Element {
	for _, item := range attrs.Items() {
		e.Attr(item.Key, item.Value)
	}
	return e
}

// ID sets the element's id. The live package addresses elements by id, so any
// element the server intends to patch needs one.
func (e *Element) ID(id string) *Element {
	return e.Attr("id", id)
}

// Class appends CSS class names. Empty names are ignored.
func (e *Element) Class(names ...string) *Element {
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			e.classes = append(e.classes, n)
		}
	}
	return e
}

// ClassIf appends the names only when cond holds.
func (e *Element) ClassIf(cond bool, names ...string) *Element {
	if cond {
		return e.Class(names...)
	}
	return e
}

// Style adds one declaration to the element's style attribute. Both the
// property and the value are checked; anything that could escape the
// declaration is an error rather than a silent substitution.
func (e *Element) Style(property, value string) *Element {
	p, v, err := sanitizeDeclaration(property, value)
	if err != nil {
		e.fail(fmt.Errorf("style on <%s>: %w", e.tag, err))
		return e
	}
	return e.set(&e.styles, p, v)
}

// Var sets a CSS custom property on the element, which is how a themed
// alacris component is configured from outside its shadow root. A leading
// "--" is optional.
//
// This is the supported way to push a runtime value into a component's
// styling; rebuilding a stylesheet per state change is the thing to avoid.
func (e *Element) Var(name, value string) *Element {
	if !strings.HasPrefix(name, "--") {
		name = "--" + name
	}
	return e.Style(name, value)
}

// Vars sets several custom properties at once, in sorted key order.
func (e *Element) Vars(vars map[string]string) *Element {
	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		e.Var(name, vars[name])
	}
	return e
}

// Apply sets custom properties from a component's theming contract, rejecting
// anything the contract does not declare. It is Vars with the safety net:
//
//	@ui.Chip(props).Apply(ui.ChipVars, map[string]string{"--chip-bg": "#ffe9a8"})
func (e *Element) Apply(vars VarSet, values map[string]string) *Element {
	return vars.Apply(e, values)
}

// On forwards a CustomEvent the component emits to a named server action,
// handled by the live package. It renders as a data-ala-on attribute; without
// the live client script on the page it does nothing.
func (e *Element) On(event, action string) *Element {
	if strings.ContainsAny(event, " \t\n:") || event == "" {
		e.fail(fmt.Errorf("event %q on <%s>: event names cannot be empty or contain spaces or colons", event, e.tag))
		return e
	}
	if strings.ContainsAny(action, " \t\n") || action == "" {
		e.fail(fmt.Errorf("action %q on <%s>: action names cannot be empty or contain spaces", action, e.tag))
		return e
	}
	e.events = append(e.events, event+":"+action)
	return e
}

// Slot renders c as light-DOM children inside a <div slot="name">, filling the
// component's named slot.
//
// The wrapper element is real and participates in layout. When that matters,
// use SlotAs to pick the tag, or put the slot attribute on your own markup —
// generated packages export the slot names as constants for exactly that.
func (e *Element) Slot(name string, c templ.Component) *Element {
	return e.SlotAs("div", name, c)
}

// SlotAs is Slot with a caller-chosen wrapper tag.
func (e *Element) SlotAs(tag, name string, c templ.Component) *Element {
	if err := ValidElementName(tag); err != nil {
		e.fail(fmt.Errorf("slot wrapper on <%s>: %w", e.tag, err))
		return e
	}
	open := "<" + tag + ` slot="` + templ.EscapeString(name) + `">`
	closing := "</" + tag + ">"
	e.slots = append(e.slots, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, open); err != nil {
			return err
		}
		if c != nil {
			if err := c.Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, closing)
		return err
	}))
	return e
}

// Children sets the element's light-DOM children explicitly, instead of taking
// them from the templ block that encloses the element.
func (e *Element) Children(c templ.Component) *Element {
	e.children = c
	return e
}

// Text sets a plain text child, escaped.
func (e *Element) Text(s string) *Element {
	return e.Children(templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, templ.EscapeString(s))
		return err
	}))
}

// set replaces an existing entry with the same key, keeping first-set order,
// so repeat calls are a last-write-wins update rather than a duplicate.
func (e *Element) set(dst *[]kv, key, value string) *Element {
	for i := range *dst {
		if (*dst)[i].key == key {
			(*dst)[i].value = value
			(*dst)[i].bare = false
			return e
		}
	}
	*dst = append(*dst, kv{key: key, value: value})
	return e
}

func (e *Element) setBare(dst *[]kv, key string) *Element {
	for i := range *dst {
		if (*dst)[i].key == key {
			(*dst)[i].value = ""
			(*dst)[i].bare = true
			return e
		}
	}
	*dst = append(*dst, kv{key: key, bare: true})
	return e
}

// Render writes the element. Props come first, then class and style, then any
// remaining attributes, so output is stable and diffable.
func (e *Element) Render(ctx context.Context, w io.Writer) error {
	if err := e.Err(); err != nil {
		return err
	}

	var b strings.Builder
	b.WriteByte('<')
	b.WriteString(e.tag)
	for _, p := range e.props {
		writeAttr(&b, p)
	}
	if len(e.classes) > 0 {
		b.WriteString(` class="`)
		b.WriteString(templ.EscapeString(strings.Join(e.classes, " ")))
		b.WriteByte('"')
	}
	if len(e.styles) > 0 {
		b.WriteString(` style="`)
		for i, s := range e.styles {
			if i > 0 {
				b.WriteByte(';')
			}
			b.WriteString(templ.EscapeString(s.key))
			b.WriteByte(':')
			b.WriteString(templ.EscapeString(s.value))
		}
		b.WriteByte('"')
	}
	for _, a := range e.attrs {
		writeAttr(&b, a)
	}
	if len(e.events) > 0 {
		b.WriteString(` data-ala-on="`)
		b.WriteString(templ.EscapeString(strings.Join(e.events, " ")))
		b.WriteByte('"')
	}
	b.WriteByte('>')

	if _, err := io.WriteString(w, b.String()); err != nil {
		return err
	}

	// templ.GetChildren never returns nil — it falls back to NopComponent —
	// so the children of the enclosing block are always safe to render.
	children := e.children
	if children == nil {
		children = templ.GetChildren(ctx)
	}
	ctx = templ.ClearChildren(ctx)
	for _, s := range e.slots {
		if err := s.Render(ctx, w); err != nil {
			return err
		}
	}
	if err := children.Render(ctx, w); err != nil {
		return err
	}

	// Custom elements are never void; the closing tag is required.
	_, err := io.WriteString(w, "</"+e.tag+">")
	return err
}

func writeAttr(b *strings.Builder, a kv) {
	b.WriteByte(' ')
	b.WriteString(a.key)
	if a.bare {
		return
	}
	b.WriteString(`="`)
	b.WriteString(templ.EscapeString(a.value))
	b.WriteByte('"')
}
