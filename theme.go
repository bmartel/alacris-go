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

// Theme is the Go form of @alacris/ui's applyTheme config. A zero Theme with
// Config.UI still applies Material Design 3 defaults: seed #e8ad18, Google
// Sans Flex, density 0, and light/dark following the OS.
//
// Re-theming is one stylesheet write. Every component consumes system tokens,
// so a new seed (or an explicit primary) re-skins the page without touching
// a component.
type Theme struct {
	// Seed is the single colour a whole scheme is grown from. Ignored for a
	// role that Colors names explicitly.
	Seed string

	// Colors names key palettes directly. Any subset is fine; omitted roles
	// are derived from Seed (or the Material default).
	Colors ThemeColors

	// Typography is a preset name ("google-sans-flex", "google-sans",
	// "roboto", "system") or a CSS font family. Empty keeps Google Sans Flex.
	Typography string

	// Radius multiplies the Material shape scale. nil keeps 1; 0 is square;
	// 2 is extra round. A pointer so 0 is distinct from "unset".
	Radius *float64

	// Motion multiplies durations. nil keeps 1; 0 is instant.
	Motion *float64

	// Density is 0, -1 or -2. nil keeps 0.
	Density *int

	// Scheme pins light or dark, or "auto" (the default) to follow the OS.
	Scheme string

	// LoadFonts is whether applyTheme injects the theme's typeface stylesheet.
	// nil means yes. Set to false when faces are self-hosted or already on
	// the page.
	LoadFonts *bool

	// Overrides are raw token writes, applied last. Keys are token names
	// without the `--ui-` prefix (`color-primary`, `radius-md`).
	Overrides ThemeOverrides
}

// ThemeColors is the explicit-palette form of Theme.Seed. Empty fields are
// derived.
type ThemeColors struct {
	Primary, Secondary, Tertiary  string
	Neutral, NeutralVariant       string
	Error, Success, Warning, Info string
}

// ThemeOverrides are last-write token maps, matching createTheme's
// `overrides: { common, light, dark }`.
type ThemeOverrides struct {
	Common map[string]string
	Light  map[string]string
	Dark   map[string]string
}

func (t Theme) scheme() string { return strings.ToLower(strings.TrimSpace(t.Scheme)) }

// wire is the JSON shape applyTheme / createTheme accept.
func (t Theme) wire() map[string]any {
	out := map[string]any{}
	if t.Seed != "" {
		out["seed"] = t.Seed
	}
	if colors := t.Colors.wire(); len(colors) > 0 {
		out["colors"] = colors
	}
	if t.Typography != "" {
		out["typography"] = t.Typography
	}
	if t.Radius != nil {
		out["shape"] = map[string]float64{"radius": *t.Radius}
	}
	if t.Motion != nil {
		out["motion"] = map[string]float64{"scale": *t.Motion}
	}
	if t.Density != nil {
		out["density"] = *t.Density
	}
	if t.LoadFonts != nil && !*t.LoadFonts {
		out["loadFonts"] = false
	}
	if ov := t.Overrides.wire(); len(ov) > 0 {
		out["overrides"] = ov
	}
	return out
}

func (c ThemeColors) wire() map[string]string {
	out := map[string]string{}
	put := func(k, v string) {
		if v != "" {
			out[k] = v
		}
	}
	put("primary", c.Primary)
	put("secondary", c.Secondary)
	put("tertiary", c.Tertiary)
	put("neutral", c.Neutral)
	put("neutralVariant", c.NeutralVariant)
	put("error", c.Error)
	put("success", c.Success)
	put("warning", c.Warning)
	put("info", c.Info)
	return out
}

func (o ThemeOverrides) wire() map[string]map[string]string {
	out := map[string]map[string]string{}
	if len(o.Common) > 0 {
		out["common"] = o.Common
	}
	if len(o.Light) > 0 {
		out["light"] = o.Light
	}
	if len(o.Dark) > 0 {
		out["dark"] = o.Dark
	}
	return out
}
