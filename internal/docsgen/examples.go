package main

import (
	"time"

	"github.com/a-h/templ"
	alacris "github.com/bmartel/alacris-go"
	"github.com/bmartel/alacris-go/ui"
)

// Every example on the documentation site lives in this file.
//
// docsgen extracts each function's body straight from this source and runs it,
// so the code a reader sees and the HTML they see beside it come from the same
// place. An example cannot drift from the library: if it stops compiling, the
// build stops; if its output changes, the committed JSON changes and the test
// that guards it fails.
//
// Add one by writing a function here and registering it in catalogue below.

// --8<-- rendering ----------------------------------------------------------

func exampleElement() templ.Component {
	return alacris.E("user-card").
		Prop("name", "Ada").
		Prop("age", 36).
		Prop("tags", []string{"math", "code"})
}

func exampleSlots() templ.Component {
	return alacris.E("info-card").
		Prop("tone", "warn").
		Slot("title", text("Ada Lovelace")).
		SlotAs("p", "body", text("Wrote the first algorithm."))
}

func exampleAttrs() templ.Component {
	return alacris.E("user-card").
		ID("ada").
		Class("card", "card--wide").
		Attr("hidden", false).
		Attr("aria-label", "A person").
		Attr("data-testid", "user-card")
}

func exampleTheming() templ.Component {
	chip := alacris.Vars("--chip-bg", "--chip-fg", "--chip-radius")
	return alacris.E("ala-chip").
		Apply(chip, map[string]string{
			"--chip-bg":     "#ffe9a8",
			"--chip-radius": "999px",
		}).
		Text("new")
}

// --8<-- encoding -----------------------------------------------------------

func exampleEncodingScalars() templ.Component {
	return alacris.E("x-demo").
		Prop("label", "hello").
		Prop("count", 42).
		Prop("ratio", 1.5).
		Prop("open", true).
		Prop("shut", false)
}

func exampleEncodingComposites() templ.Component {
	type point struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	return alacris.E("x-demo").
		Prop("tags", []string{"a", "b"}).
		Prop("origin", point{X: 1, Y: 2}).
		Prop("lookup", map[string]int{"b": 2, "a": 1}).
		Prop("at", time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)).
		Prop("timeout", 1500*time.Millisecond)
}

func exampleEncodingOmitted() templ.Component {
	// A nil value leaves the attribute off entirely, so the component keeps
	// the default it declared in define().
	return alacris.E("x-demo").
		Prop("name", nil).
		Prop("tags", []string(nil)).
		Prop("kept", "written")
}

func exampleEncodingEscaping() templ.Component {
	// Interpolated values are attribute values, never markup. There is
	// nothing to escape by hand and no way to forget.
	return alacris.E("x-demo").
		Prop("label", `" onload="alert(1)`).
		Prop("payload", map[string]string{"html": "</x-demo><script>"})
}

// --8<-- page setup ---------------------------------------------------------

func exampleScripts() templ.Component {
	return alacris.Scripts(alacris.Config{
		Modules: []string{"/web/components.js"},
		Version: "2026.08.08",
	})
}

func exampleScriptsLive() templ.Component {
	return alacris.Scripts(alacris.Config{
		Modules: []string{"/web/components.js"},
		Live:    true,
		Page:    "PAGE-ID",
	})
}

func examplePending() templ.Component {
	return alacris.Pending{Tags: []string{"user-card", "ala-chip"}}
}

func exampleUIButton() templ.Component {
	return ui.Button(ui.ButtonProps{Variant: "tonal"}).Text("Save")
}

func exampleUITheme() templ.Component {
	return alacris.Scripts(alacris.Config{
		UI: true,
		Theme: alacris.Theme{
			Seed:   "#0b57d0",
			Scheme: "dark",
		},
	})
}

// --8<-- generated wrappers -------------------------------------------------

func exampleGenerated() templ.Component {
	// What ui.TodoList(...) expands to. The generator knows every prop's
	// default, so Filter: "all" would have been left off the element.
	return alacris.E("ala-todo-list").
		ID("todos").
		Prop("items", []todoItem{{ID: 1, Text: "Read the docs"}}).
		Prop("filter", "active").
		On("add", "add-todo").
		On("toggle", "toggle-todo")
}

type todoItem struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// text is a templ.Component that writes a literal string.
func text(s string) templ.Component {
	return templ.Raw(templ.EscapeString(s))
}

// catalogue is every example, in the order they are most useful to read.
//
// The id is what a page passes to <GoExample id="..." />.
var catalogue = []example{
	{id: "element", fn: "exampleElement", tags: []string{"user-card"},
		note: "Props are kebab-cased and encoded by their Go type."},
	{id: "slots", fn: "exampleSlots", tags: []string{"info-card"},
		note: "Slot content is light DOM, so it is in the first paint."},
	{id: "attrs", fn: "exampleAttrs", tags: []string{"user-card"},
		note: "Ordinary attributes follow HTML rules: a false bool disappears."},
	{id: "theming", fn: "exampleTheming", tags: []string{"ala-chip"},
		note: "Custom properties reach through the shadow boundary."},

	{id: "encoding-scalars", fn: "exampleEncodingScalars", tags: []string{"x-demo"},
		note: "Booleans are always written out, never signalled by presence."},
	{id: "encoding-composites", fn: "exampleEncodingComposites", tags: []string{"x-demo"},
		note: "Objects and arrays cross as JSON, so no post-load assignment is needed."},
	{id: "encoding-omitted", fn: "exampleEncodingOmitted", tags: []string{"x-demo"},
		note: "A nil prop leaves the component's own default alone."},
	{id: "encoding-escaping", fn: "exampleEncodingEscaping", tags: []string{"x-demo"},
		note: "Values are attribute values, never markup."},

	{id: "scripts", fn: "exampleScripts",
		note: "The import map has to precede the first module import."},
	{id: "scripts-live", fn: "exampleScriptsLive",
		note: "The live client carries its endpoint and session on its script tag."},
	{id: "pending", fn: "examplePending",
		note: "Hides elements until their module has defined them."},

	{id: "generated", fn: "exampleGenerated", tags: []string{"ala-todo-list"},
		note: "What a generated wrapper expands to."},

	{id: "ui-button", fn: "exampleUIButton", tags: []string{"ui-button"},
		note: "Every Alacris UI tag has a typed wrapper in github.com/bmartel/alacris-go/ui."},
	{id: "ui-theme", fn: "exampleUITheme",
		note: "Config.UI loads the design system; Theme re-skins every component at once."},
}

// render dispatches an example by function name. It is written out by hand
// rather than reflected so that an example that is registered but not wired up
// fails to compile.
func render(fn string) templ.Component {
	switch fn {
	case "exampleElement":
		return exampleElement()
	case "exampleSlots":
		return exampleSlots()
	case "exampleAttrs":
		return exampleAttrs()
	case "exampleTheming":
		return exampleTheming()
	case "exampleEncodingScalars":
		return exampleEncodingScalars()
	case "exampleEncodingComposites":
		return exampleEncodingComposites()
	case "exampleEncodingOmitted":
		return exampleEncodingOmitted()
	case "exampleEncodingEscaping":
		return exampleEncodingEscaping()
	case "exampleScripts":
		return exampleScripts()
	case "exampleScriptsLive":
		return exampleScriptsLive()
	case "examplePending":
		return examplePending()
	case "exampleGenerated":
		return exampleGenerated()
	case "exampleUIButton":
		return exampleUIButton()
	case "exampleUITheme":
		return exampleUITheme()
	}
	return nil
}
