package gen

import (
	"strings"
	"testing"
)

// The lexer's real job is knowing where a token ends. Everything here is
// source that would make a naive scanner read code as data.
func TestLexerSkipsCodeThatLooksLikeData(t *testing.T) {
	src := "" +
		"const re = /define\\('fake-tag', \\{ props: \\{ x: 1 \\} \\}\\)/g;\n" +
		"const s  = 'define(\"also-fake\", { props: { y: 2 } })';\n" +
		"const t  = `define(\"template-fake\", ${ `${ nested }` } { props: { z: 3 } })`;\n" +
		"const q  = `url(\"${src.replace(/\"/g, '%22')}\")`;\n" +
		"// define('comment-fake', { props: { a: 1 } })\n" +
		"/* define('block-fake', { props: { b: 1 } }) */\n" +
		"const div = a / b / c;\n" +
		"define('real-one', { props: { n: 1 }, setup });\n"

	got, err := ParseSource("t.js", src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(got) != 1 || got[0].Tag != "real-one" {
		var tags []string
		for _, c := range got {
			tags = append(tags, c.Tag)
		}
		t.Fatalf("found %v, want only real-one", tags)
	}
}

func TestLexerSkipsRegexQuoteInsideTemplateSubstitution(t *testing.T) {
	src := "import { define, html, css } from 'alacris';\n" +
		"export function ghostFor(src) {\n" +
		"  return `background-image:url(\"${src.replace(/\"/g, '%22')}\");`;\n" +
		"}\n" +
		"define('x-b', { props: { src: '' }, setup({ src }) {\n" +
		"  return html`<div style=${() => ghostFor(src())}></div>`;\n" +
		"} });\n"
	got, err := ParseSource("web/b.js", src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(got) != 1 || got[0].Tag != "x-b" {
		t.Fatalf("found %+v, want x-b", got)
	}
}

func TestParseProps(t *testing.T) {
	src := `
define('user-card', {
  props: {
    name: 'anon',
    age: 0,
    ratio: 1.5,
    active: true,
    pinned: false,
    tags: [],
    labels: ['a', 'b'],
    sizes: [1, 2],
    config: { deep: true },
    maxCount: 10,
    nothing: null,
  },
  styles: css` + "`" + `:host { display: block }` + "`" + `,
  setup({ name }, host) { return html` + "`" + `<p>${name}</p>` + "`" + ` },
});
`
	cs, err := ParseSource("user-card.js", src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(cs) != 1 {
		t.Fatalf("found %d components, want 1", len(cs))
	}

	want := map[string]struct {
		kind    Kind
		goType  string
		attr    string
		defualt string
	}{
		"name":     {KindString, "string", "name", `"anon"`},
		"age":      {KindNumber, "int", "age", "0"},
		"ratio":    {KindNumber, "float64", "ratio", "1.5"},
		"active":   {KindBool, "bool", "active", "true"},
		"pinned":   {KindBool, "bool", "pinned", "false"},
		"tags":     {KindJSON, "[]any", "tags", ""},
		"labels":   {KindJSON, "[]string", "labels", ""},
		"sizes":    {KindJSON, "[]int", "sizes", ""},
		"config":   {KindJSON, "map[string]any", "config", ""},
		"maxCount": {KindNumber, "int", "max-count", "10"},
		"nothing":  {KindJSON, "any", "nothing", ""},
	}

	if len(cs[0].Props) != len(want) {
		t.Fatalf("found %d props, want %d", len(cs[0].Props), len(want))
	}
	for _, p := range cs[0].Props {
		w, ok := want[p.Name]
		if !ok {
			t.Errorf("unexpected prop %q", p.Name)
			continue
		}
		if p.Kind != w.kind {
			t.Errorf("%s: kind %q, want %q", p.Name, p.Kind, w.kind)
		}
		if p.GoType != w.goType {
			t.Errorf("%s: Go type %q, want %q", p.Name, p.GoType, w.goType)
		}
		if p.Attr != w.attr {
			t.Errorf("%s: attribute %q, want %q", p.Name, p.Attr, w.attr)
		}
		if w.defualt != "" && p.GoDefault != w.defualt {
			t.Errorf("%s: Go default %q, want %q", p.Name, p.GoDefault, w.defualt)
		}
	}
}

func TestParseJSDoc(t *testing.T) {
	src := `
/**
 * A person, at a glance.
 *
 * @prop {string[]} tags the labels shown under the name
 * @prop {integer} age
 * @prop name what to call them
 * @fires greet {name: string, times: number} - the user said hello
 * @fires reset - the card went back to its initial state
 * @slot title - replaces the heading
 * @slot - anything else
 * @goname PersonCard
 */
define('user-card', {
  props: { name: 'anon', age: 0, tags: [] },
  setup,
});
`
	cs, err := ParseSource("user-card.js", src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	c := cs[0]

	if c.Doc != "A person, at a glance." {
		t.Errorf("doc = %q", c.Doc)
	}
	if c.GoName != "PersonCard" {
		t.Errorf("goName = %q", c.GoName)
	}

	byName := map[string]Prop{}
	for _, p := range c.Props {
		byName[p.Name] = p
	}
	if got := byName["tags"].GoType; got != "[]string" {
		t.Errorf("tags type = %q, want []string (the JSDoc has to beat the empty array)", got)
	}
	if got := byName["age"].GoType; got != "int" {
		t.Errorf("age type = %q, want int", got)
	}
	if got := byName["name"].Doc; got != "what to call them" {
		t.Errorf("name doc = %q", got)
	}

	if len(c.Events) != 2 {
		t.Fatalf("found %d events, want 2", len(c.Events))
	}
	greet := c.Events[0]
	if greet.Name != "greet" || greet.Doc != "the user said hello" {
		t.Errorf("greet = %+v", greet)
	}
	if len(greet.Detail) != 2 ||
		greet.Detail[0] != (Field{Name: "name", GoType: "string"}) ||
		greet.Detail[1] != (Field{Name: "times", GoType: "float64"}) {
		t.Errorf("greet detail = %+v", greet.Detail)
	}
	if c.Events[1].Name != "reset" || len(c.Events[1].Detail) != 0 {
		t.Errorf("reset = %+v", c.Events[1])
	}

	if len(c.Slots) != 2 || c.Slots[0].Name != "title" || c.Slots[1].Name != "" {
		t.Errorf("slots = %+v", c.Slots)
	}
	if c.Slots[0].Doc != "replaces the heading" {
		t.Errorf("slot doc = %q", c.Slots[0].Doc)
	}
}

func TestParseShorthandAndShadow(t *testing.T) {
	src := `
define('x-hello', () => html` + "`" + `<p>hello</p>` + "`" + `);
define('x-light', { shadow: false, setup: () => html` + "`" + `<p>x</p>` + "`" + ` });
define('x-closed', { shadow: 'closed', setup });
`
	cs, err := ParseSource("t.js", src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(cs) != 3 {
		t.Fatalf("found %d components, want 3", len(cs))
	}
	if cs[0].Tag != "x-hello" || len(cs[0].Props) != 0 {
		t.Errorf("shorthand: %+v", cs[0])
	}
	if cs[1].Shadow != "none" {
		t.Errorf("shadow: false should read as %q, got %q", "none", cs[1].Shadow)
	}
	if cs[2].Shadow != "closed" {
		t.Errorf("shadow = %q", cs[2].Shadow)
	}
}

func TestParseIgnoresOtherDefines(t *testing.T) {
	src := `
define('amd/module', ['dep'], function (dep) { return {}; });
customElements.define('x-native', class extends HTMLElement {});
someLib.define('x-other', { unrelated: 1 });
define('x-real', { setup });
`
	cs, err := ParseSource("t.js", src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(cs) != 1 || cs[0].Tag != "x-real" {
		var tags []string
		for _, c := range cs {
			tags = append(tags, c.Tag)
		}
		t.Fatalf("found %v, want only x-real", tags)
	}
}

func TestParseRejectsWhatItCannotRead(t *testing.T) {
	cases := map[string]string{
		"props built at runtime": `define('x-y', { props: buildProps(), setup });`,
		"spread into props":      `define('x-y', { props: { ...base, a: 1 }, setup });`,
		"computed default":       `define('x-y', { props: { a: compute() }, setup });`,
		"prop documented but not declared": `
/** @prop {string} missing */
define('x-y', { props: { a: 1 }, setup });`,
		"unknown jsdoc type": `
/** @prop {Widget} a */
define('x-y', { props: { a: 1 }, setup });`,
		"scalar type for a json prop": `
/** @prop {string} a */
define('x-y', { props: { a: [] }, setup });`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseSource("t.js", src)
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			if !strings.Contains(err.Error(), "t.js") {
				t.Errorf("error should say where: %v", err)
			}
		})
	}
}

func TestParseUIFileHeader(t *testing.T) {
	src := `
// <ui-button> — the Material common button.
//
// @prop  {string}  variant='filled' — filled | tonal | outlined
// @prop  {boolean} disabled=false
// @event (native click bubbles; no custom event)
// @event change — pressed; detail: { value }
// @slot  (default) — label
// @slot  icon, trailing — icons
//
import { define, vars } from '@alacris/core';

const t = vars('ui-button', {
  height: '40px',
  filledBg: sys.color.primary,
});

define('ui-button', {
  props: { variant: 'filled', disabled: false },
  setup,
});
`
	cs, err := ParseSource("ui-button.js", src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(cs) != 1 {
		t.Fatalf("found %d components, want 1", len(cs))
	}
	c := cs[0]
	if !strings.Contains(c.Doc, "Material common button") {
		t.Errorf("doc = %q", c.Doc)
	}
	byName := map[string]Prop{}
	for _, p := range c.Props {
		byName[p.Name] = p
	}
	if got := byName["variant"].Doc; !strings.Contains(got, "filled") {
		t.Errorf("variant doc = %q", byName["variant"].Doc)
	}
	if len(c.Events) != 1 || c.Events[0].Name != "change" {
		t.Errorf("events = %+v", c.Events)
	}
	if len(c.Events[0].Detail) != 1 || c.Events[0].Detail[0].Name != "value" {
		t.Errorf("change detail = %+v", c.Events[0].Detail)
	}
	if len(c.Slots) != 3 || c.Slots[0].Name != "" || c.Slots[1].Name != "icon" || c.Slots[2].Name != "trailing" {
		t.Errorf("slots = %+v", c.Slots)
	}
	wantVars := map[string]bool{"--ui-button-height": true, "--ui-button-filled-bg": true}
	if len(c.CSSProps) != 2 {
		t.Errorf("css props = %+v", c.CSSProps)
	}
	for _, p := range c.CSSProps {
		if !wantVars[p.Name] {
			t.Errorf("unexpected css prop %q", p.Name)
		}
	}
}

func TestParseDocAssociation(t *testing.T) {
	// A doc comment belongs to what follows it, and only until the next
	// statement boundary.
	src := `
/** Belongs to the helper, not to the component. */
function helper() { return 1; }

define('x-orphan', { setup });

/** Belongs to this one. */
export const Card = define('x-card', { setup });
`
	cs, err := ParseSource("t.js", src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	byTag := map[string]Component{}
	for _, c := range cs {
		byTag[c.Tag] = c
	}
	if got := byTag["x-orphan"].Doc; got != "" {
		t.Errorf("x-orphan picked up a stray doc comment: %q", got)
	}
	if got := byTag["x-card"].Doc; got != "Belongs to this one." {
		t.Errorf("x-card doc = %q", got)
	}
}

func TestParseDocBlockDoesNotBindAcrossAnInterveningDefine(t *testing.T) {
	src := `
define('x-first', {
  props: {},
  setup() { return html` + "`" + `<span>first</span>` + "`" + `; },
});

/**
 * The grid.
 *
 * @prop {integer} rows how many rows
 */
define('x-second', {
  props: { rows: 0 },
  setup({ rows }) { return html` + "`" + `<span>${rows}</span>` + "`" + `; },
});
`
	cs, err := ParseSource("web/c.js", src)
	if err != nil {
		t.Fatalf("adjacent doc should generate: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("found %d components, want 2", len(cs))
	}

	moved := `
/**
 * The grid.
 *
 * @prop {integer} rows how many rows
 */
define('x-first', {
  props: {},
  setup() { return html` + "`" + `<span>first</span>` + "`" + `; },
});

define('x-second', {
  props: { rows: 0 },
  setup({ rows }) { return html` + "`" + `<span>${rows}</span>` + "`" + `; },
});
`
	_, err = ParseSource("web/c.js", moved)
	if err == nil {
		t.Fatal("expected an error when the doc block is stolen by x-first")
	}
	msg := err.Error()
	if !strings.Contains(msg, "doc block documents props not on <x-first>") {
		t.Errorf("error should name the wrong component: %v", err)
	}
	if !strings.Contains(msg, "did you mean <x-second>") {
		t.Errorf("error should point at the documented component: %v", err)
	}
}

func TestExportedName(t *testing.T) {
	cases := map[string]string{
		"user-card":  "UserCard",
		"ala-todos":  "AlaTodos",
		"maxCount":   "MaxCount",
		"aria-label": "AriaLabel",
		"url":        "URL",
		"avatarUrl":  "AvatarURL",
		"itemId":     "ItemID",
		"x":          "X",
		"data_value": "DataValue",
	}
	for in, want := range cases {
		if got := exportedName(in); got != want {
			t.Errorf("exportedName(%q) = %q, want %q", in, got, want)
		}
	}
}

// Generated file names come from source file names, which come from a
// directory walk. Nothing hostile is expected there, but a name that escaped
// the output directory would be a bad surprise, and the check is one test.
func TestGeneratedFileNamesStayInsideTheOutputDirectory(t *testing.T) {
	nasty := []string{
		"../escape.js",
		"../../etc/passwd.js",
		`..\windows.js`,
		"/absolute.js",
		"a/b/c.js",
		"...js",
		"....js",
		`C:\evil.js`,
		"",
		"---.js",
	}

	for _, name := range nasty {
		base := fileBase(name)
		if strings.ContainsAny(base, `/\:`) || strings.Contains(base, "..") || base == "" {
			t.Errorf("fileBase(%q) = %q, which is not a plain file name", name, base)
		}
	}
}

// Two components whose sources have the same base name in different
// directories would otherwise overwrite each other's generated file.
func TestGeneratedFileNamesAreDerivedFromTheBaseName(t *testing.T) {
	cases := map[string]string{
		"user-card.js":     "user_card",
		"web/user-card.js": "user_card",
		"UserCard.mjs":     "user_card",
		"2fa.js":           "x2fa",
	}
	for in, want := range cases {
		if got := fileBase(in); got != want {
			t.Errorf("fileBase(%q) = %q, want %q", in, got, want)
		}
	}
}
