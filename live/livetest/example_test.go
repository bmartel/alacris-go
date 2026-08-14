package livetest_test

import (
	"fmt"
	"testing"

	"github.com/bmartel/alacris-go/live"
	"github.com/bmartel/alacris-go/live/livetest"
)

// The shape of a handler test: register the code under test, invoke the
// action, assert on the patches — no HTTP server, no SSE parsing.
func Example() {
	t := &testing.T{} // in a real test this is the *testing.T you were given

	srv := live.New()
	defer srv.Close()
	live.On(srv, "add-todo", func(c *live.Ctx, d struct {
		Text string `json:"text"`
	}) error {
		c.Session.Element("todos").Set("items", []string{d.Text})
		return nil
	})

	sess, rec := livetest.NewSession(t, srv)
	_ = livetest.Invoke(t, sess, "add-todo", "todos", map[string]string{"text": "milk"})

	p, ok := rec.Last("todos", "items")
	fmt.Println(ok, p.Key)
	// Output: true items
}
