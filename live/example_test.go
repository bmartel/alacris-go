package live_test

import (
	"net/http"

	"github.com/bmartel/alacris-go/live"
)

// On binds a named action to a handler whose detail is already decoded into
// the type the generated wrappers declare for the event.
func ExampleOn() {
	srv := live.New()
	defer srv.Close()

	type addDetail struct {
		Text string `json:"text"`
	}
	live.On(srv, "add-todo", func(c *live.Ctx, d addDetail) error {
		// One property write is one DOM update on the page.
		c.Session.Element("todos").Set("items", []string{d.Text})
		return nil
	})
}

// NewSession runs during the page render, before anything is written to the
// response, because it may need to set the session cookie.
func ExampleServer_NewSession() {
	srv := live.New()
	defer srv.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		sess := srv.NewSession(w, r) // before the first byte of the body

		// Register OnOpen and push the full state from it: a reconnecting
		// EventSource missed everything sent while it was away, and this runs
		// on every attach, including reconnects.
		sess.OnOpen(func(s *live.Session) {
			s.Element("cart").Set("count", 0)
		})

		// Render the page with alacris.Config{Live: true, Page: sess.ID()}.
	})
}
