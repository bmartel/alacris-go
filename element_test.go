package alacris

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var b bytes.Buffer
	if err := c.Render(templ.InitializeContext(context.Background()), &b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

func renderErr(t *testing.T, c templ.Component) error {
	t.Helper()
	return c.Render(templ.InitializeContext(context.Background()), io.Discard)
}

func text(s string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

func TestElementRender(t *testing.T) {
	cases := []struct {
		name string
		el   *Element
		want string
	}{
		{
			"props are kebab-cased and typed",
			E("user-card").Prop("name", "Ada").Prop("maxCount", 3).Prop("tags", []string{"a", "b"}),
			`<user-card name="Ada" max-count="3" tags="[&#34;a&#34;,&#34;b&#34;]"></user-card>`,
		},
		{
			"a nil prop leaves the component's default alone",
			E("user-card").Prop("name", nil).Prop("age", 1),
			`<user-card age="1"></user-card>`,
		},
		{
			"props keep first-set order and last-set value",
			E("x-y").Prop("b", 1).Prop("a", 1).Prop("b", 2),
			`<x-y b="2" a="1"></x-y>`,
		},
		{
			"class accumulates, style is sanitized",
			E("x-y").Class("card", "wide").ClassIf(false, "hidden").Var("chip-bg", "#111"),
			`<x-y class="card wide" style="--chip-bg:#111"></x-y>`,
		},
		{
			"html attributes follow html rules",
			E("x-y").ID("cart").Attr("hidden", true).Attr("disabled", false).Attr("aria-label", "Cart"),
			`<x-y id="cart" hidden aria-label="Cart"></x-y>`,
		},
		{
			"events render as one delegated declaration",
			E("x-y").On("greet", "say-hi").On("reset", "clear"),
			`<x-y data-ala-on="greet:say-hi reset:clear"></x-y>`,
		},
		{
			"debounce sets data-ala-debounce in milliseconds",
			E("x-y").On("input", "search").Debounce(150 * time.Millisecond),
			`<x-y data-ala-debounce="150" data-ala-on="input:search"></x-y>`,
		},
		{
			"slots wrap their content",
			E("x-y").Slot("title", text("<b>hi</b>")).SlotAs("span", "badge", text("new")),
			`<x-y><div slot="title"><b>hi</b></div><span slot="badge">new</span></x-y>`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := render(t, c.el); got != c.want {
				t.Errorf("\n got: %s\nwant: %s", got, c.want)
			}
		})
	}
}

func TestElementEscapesAttributeValues(t *testing.T) {
	// A prop value is arbitrary user data. It must not be able to close the
	// attribute, and JSON props are the case most likely to carry quotes.
	got := render(t, E("x-y").
		Prop("label", `" onload="alert(1)`).
		Prop("data", map[string]string{"k": `</x-y><script>`}))

	for _, bad := range []string{`onload="alert`, "<script>", "</x-y><"} {
		if strings.Contains(strings.TrimSuffix(got, "</x-y>"), bad) {
			t.Errorf("unescaped %q in output: %s", bad, got)
		}
	}
	if !strings.Contains(got, "&#34;") {
		t.Errorf("quotes were not escaped: %s", got)
	}
}

func TestElementChildren(t *testing.T) {
	t.Run("from the enclosing templ block", func(t *testing.T) {
		ctx := templ.WithChildren(templ.InitializeContext(context.Background()), text("slotted"))
		var b bytes.Buffer
		if err := E("x-y").Render(ctx, &b); err != nil {
			t.Fatal(err)
		}
		if got, want := b.String(), `<x-y>slotted</x-y>`; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("children are not passed on to the block twice", func(t *testing.T) {
		// Without ClearChildren, a nested element would render the outer
		// block's children again inside itself.
		inner := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			var nested bytes.Buffer
			if err := templ.GetChildren(ctx).Render(ctx, &nested); err != nil {
				return err
			}
			if nested.Len() != 0 {
				t.Errorf("nested component still saw the outer children: %q", nested.String())
			}
			_, err := io.WriteString(w, "ok")
			return err
		})
		ctx := templ.WithChildren(templ.InitializeContext(context.Background()), inner)
		if err := E("x-y").Render(ctx, io.Discard); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("explicit text is escaped", func(t *testing.T) {
		if got, want := render(t, E("x-y").Text("<b>")), `<x-y>&lt;b&gt;</x-y>`; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})
}

func TestElementReportsBuildErrors(t *testing.T) {
	cases := map[string]*Element{
		"tag without a hyphen":  E("div"),
		"uppercase in tag":      E("userCard-x"),
		"reserved tag":          E("font-face"),
		"style attribute":       E("x-y").Attr("style", "color:red"),
		"css value escaping":    E("x-y").Var("bg", "red;position:fixed"),
		"css comment":           E("x-y").Var("bg", "red/*x*/"),
		"javascript url in css": E("x-y").Var("bg", "url(javascript:alert(1))"),
		"event with a colon":    E("x-y").On("a:b", "act"),
		"empty action":          E("x-y").On("greet", ""),
		"unsafe integer prop":   E("x-y").Prop("id", int64(MaxSafeInteger+1)),
		"attribute name":        E("x-y").Attr(`bad"name`, "x"),
	}
	for name, el := range cases {
		t.Run(name, func(t *testing.T) {
			if err := renderErr(t, el); err == nil {
				t.Error("expected an error, got none")
			}
		})
	}
}

func TestVarSet(t *testing.T) {
	chip := NewVarSet("chip", "bg", "fg", "borderRadius")

	if got, want := chip.Name("borderRadius"), "--chip-border-radius"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
	want := []string{"--chip-bg", "--chip-border-radius", "--chip-fg"}
	got := chip.Names()
	if len(got) != len(want) {
		t.Fatalf("Names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names = %v, want %v", got, want)
		}
	}

	el := chip.Apply(E("x-chip"), map[string]string{"bg": "#111", "fg": "#fff"})
	if out, want := render(t, el), `<x-chip style="--chip-bg:#111;--chip-fg:#fff"></x-chip>`; out != want {
		t.Errorf("got %s, want %s", out, want)
	}

	// A key that is not part of the contract is a rename that has not been
	// followed through, not a value to write out and hope.
	if err := renderErr(t, chip.Apply(E("x-chip"), map[string]string{"colour": "#111"})); err == nil {
		t.Error("expected an error for a key outside the contract")
	}
}

func TestPendingStyles(t *testing.T) {
	got := render(t, Pending{Tags: []string{"user-card", "ala-chip"}})
	want := `<style>user-card:not(:defined),ala-chip:not(:defined){visibility:hidden}</style>`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}

	if got := render(t, Pending{}); got != "" {
		t.Errorf("no tags should render nothing, got %s", got)
	}

	if err := renderErr(t, Pending{Tags: []string{"div"}}); err == nil {
		t.Error("expected an error for a non-custom-element tag")
	}
}

func TestScripts(t *testing.T) {
	t.Run("import map matches the npm specifiers", func(t *testing.T) {
		got := render(t, Scripts(Config{Modules: []string{"/app.js"}}))
		for _, want := range []string{
			`<script type="importmap">`,
			`"alacris":"/_alacris/alacris.js"`,
			`"alacris/store":"/_alacris/store.js"`,
			`"alacris/context":"/_alacris/context.js"`,
			`"alacris/signal":"/_alacris/signal.js"`,
			`<script type="module" src="/app.js"></script>`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %s in:\n%s", want, got)
			}
		}
		// The import map has to come first, or module imports resolve without it.
		if strings.Index(got, "importmap") > strings.Index(got, "/app.js") {
			t.Error("import map must precede module scripts")
		}
	})

	t.Run("dev build and custom base", func(t *testing.T) {
		got := render(t, Scripts(Config{Base: "/static/js", Dev: true}))
		if !strings.Contains(got, `"alacris":"/static/js/alacris.dev.js"`) {
			t.Errorf("got %s", got)
		}
	})

	t.Run("live client carries its endpoint and page id", func(t *testing.T) {
		got := render(t, Scripts(Config{Live: true, Page: "page-1"}))
		for _, want := range []string{
			`src="/_alacris/live.js"`,
			`data-endpoint="/_alacris/live"`,
			`data-page="page-1"`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %s in:\n%s", want, got)
			}
		}
	})

	t.Run("nonce reaches every script", func(t *testing.T) {
		ctx := templ.WithNonce(templ.InitializeContext(context.Background()), "n0nce")
		var b bytes.Buffer
		if err := Scripts(Config{Modules: []string{"/app.js"}, Live: true}).Render(ctx, &b); err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(b.String(), `nonce="n0nce"`); n != 3 {
			t.Errorf("nonce on %d of 3 script tags:\n%s", n, b.String())
		}
	})

	t.Run("nonce reaches the UI bootstrap too", func(t *testing.T) {
		ctx := templ.WithNonce(templ.InitializeContext(context.Background()), "n0nce")
		var b bytes.Buffer
		if err := Scripts(Config{Modules: []string{"/app.js"}, Live: true, UI: true}).Render(ctx, &b); err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(b.String(), `nonce="n0nce"`); n != 4 {
			t.Errorf("nonce on %d of 4 script tags:\n%s", n, b.String())
		}
	})

	t.Run("UI import map and bootstrap", func(t *testing.T) {
		got := render(t, Scripts(Config{UI: true, Theme: Theme{Seed: "#0b57d0", Scheme: "dark"}}))
		for _, want := range []string{
			`"@alacris/core":"/_alacris/alacris.js"`,
			`"@alacris/ui":"/_alacris/ui/index.js"`,
			`"@alacris/ui/theme":"/_alacris/ui/theme/index.js"`,
			`import { applyTheme, setScheme } from '@alacris/ui/theme'`,
			`import '@alacris/ui'`,
			`applyTheme({`,
			`"seed":"#0b57d0"`,
			`setScheme("dark")`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %s in:\n%s", want, got)
			}
		}
		// The UI module has to come after the import map and before app modules.
		if i, j := strings.Index(got, "importmap"), strings.Index(got, "@alacris/ui'"); i < 0 || j < 0 || i > j {
			t.Error("import map must precede the UI bootstrap")
		}
	})

	t.Run("import map cannot be closed by an entry", func(t *testing.T) {
		got := render(t, Scripts(Config{Imports: map[string]string{"x": `</script><script>alert(1)`}}))
		if strings.Contains(got, "</script><script>alert") {
			t.Errorf("import map escaped its element:\n%s", got)
		}
	})
}

// renderString renders an element and returns its output, for tests that care
// whether it was refused rather than what it produced.
func renderString(c templ.Component) (string, error) {
	var b bytes.Buffer
	err := c.Render(templ.InitializeContext(context.Background()), &b)
	return b.String(), err
}
