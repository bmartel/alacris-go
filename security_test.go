package alacris

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A prop name reaches the start tag directly — it is not a value and cannot be
// escaped, so it has to be refused. Names are usually literals in generated
// code, but Props takes a map, and a map is the sort of thing that ends up
// holding data.
func TestPropNamesCannotInjectAttributes(t *testing.T) {
	injections := map[string]string{
		"closes the attribute":  `a" onload="alert(1)`,
		"opens a new attribute": `a onload=alert(1)`,
		"closes the tag":        `a><script>alert(1)</script`,
		"single quotes":         `a' onload='alert(1)`,
		"equals sign":           `a=b`,
		"tag delimiter":         `a<b`,
		"empty":                 ``,
		"newline":               "a\nonload=x",
	}

	for name, bad := range injections {
		t.Run(name, func(t *testing.T) {
			for _, el := range []*Element{
				E("x-y").Prop(bad, "v"),
				E("x-y").PropRaw(bad, "v"),
				E("x-y").Props(map[string]any{bad: "v"}),
			} {
				out, err := renderString(el)
				if err != nil {
					continue // refused, which is the point
				}
				t.Errorf("prop name %q was accepted and rendered: %s", bad, out)
			}
		})
	}
}

// The same holds for a slot wrapper: the tag name is written into the start and
// end tags, so it has to be a tag name and nothing else.
func TestSlotWrapperTagsAreRestricted(t *testing.T) {
	bad := []string{
		`div<script`,
		`div onload=x`,
		`div"`,
		``,
		`1div`,
		`-div`,
		// Raw-text elements do not escape their content the way every other
		// element does, so a slot wrapper has no business being one.
		`script`,
		`style`,
		`textarea`,
		`title`,
		`iframe`,
	}

	for _, tag := range bad {
		out, err := renderString(E("x-y").SlotAs(tag, "s", text("hi")))
		if err == nil {
			t.Errorf("slot wrapper tag %q was accepted and rendered: %s", tag, out)
		}
	}

	// Ordinary elements still work.
	for _, tag := range []string{"div", "span", "p", "h3", "li", "my-wrapper"} {
		if _, err := renderString(E("x-y").SlotAs(tag, "s", text("hi"))); err != nil {
			t.Errorf("slot wrapper tag %q was refused: %v", tag, err)
		}
	}
}

// The asset handler used to remember every name it was asked for, including
// the ones that do not exist — so a few thousand requests for made-up file
// names grew a map that nothing ever emptied.
func TestRuntimeHandlerDoesNotGrowOnUnknownNames(t *testing.T) {
	h := RuntimeHandler().(*runtimeHandler)

	for i := 0; i < 2000; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/_alacris/nope-"+strings.Repeat("x", i%40)+itoa(i)+".js", nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("unknown asset returned %d", rec.Code)
		}
	}

	// Only the assets that actually exist may be held.
	if len(h.assets) != len(vendoredHashes) {
		t.Errorf("the handler holds %d names after 2000 unknown requests; it should only ever hold the %d it serves",
			len(h.assets), len(vendoredHashes))
	}
}
