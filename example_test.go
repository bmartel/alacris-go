package alacris_test

import (
	"context"
	"os"

	alacris "github.com/bmartel/alacris-go"
)

// Every prop crosses as an attribute, objects and arrays included, so the
// element is complete before any JavaScript has run.
func ExampleE() {
	card := alacris.E("user-card").
		Prop("name", "Ada").
		Prop("tags", []string{"math", "code"}).
		ID("ada")
	_ = card.Render(context.Background(), os.Stdout)
	// Output: <user-card name="Ada" tags="[&#34;math&#34;,&#34;code&#34;]" id="ada"></user-card>
}

// Scripts emits the import map and module tags a page needs; with Version set,
// every asset URL is release-specific and cacheable for a year.
func ExampleConfig_Scripts() {
	cfg := alacris.Config{
		Version: "1.0.0",
		Modules: []string{"/static/app.js"},
	}
	_ = cfg.Scripts().Render(context.Background(), os.Stdout)
	// Output: <script type="importmap">{"imports":{"alacris":"/_alacris/alacris.js?v=1.0.0","alacris/context":"/_alacris/context.js?v=1.0.0","alacris/signal":"/_alacris/signal.js?v=1.0.0","alacris/store":"/_alacris/store.js?v=1.0.0"}}</script><script type="module" src="/static/app.js"></script>
}
