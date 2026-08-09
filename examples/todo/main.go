// Command todo is the alacris-go example.
//
//	go run ./examples/todo
//	open http://localhost:8080
//
// It shows the three layers together:
//
//   - the components are written in JavaScript, in web/components.js, because
//     setup() runs in the browser;
//   - ui/ is generated from them by alacris-go, so the props are typed;
//   - the list itself lives here, in Go, and reaches the page as prop writes.
//
// Nothing in the request path renders component internals. The server renders
// elements and their attributes; alacris does the rest.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	alacris "github.com/bmartel/alacris-go"
	"github.com/bmartel/alacris-go/examples/todo/model"
	"github.com/bmartel/alacris-go/examples/todo/ui"
	"github.com/bmartel/alacris-go/live"
)

// Regenerating is two steps, in this order: the Go wrappers first, because the
// templates use them.
//
// The templ CLI is pinned by version so it runs in its own module rather than
// dragging its dependencies into this one.
//
//go:generate go run github.com/bmartel/alacris-go/cmd/alacris-go generate ./web -o ./ui -strip ala-
//go:generate go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate

//go:embed web
var webFS embed.FS

// The ids and action names the page and the handlers agree on. Constants
// because a typo in either half is otherwise a silent no-op.
const (
	todosID   = "todos"
	echoID    = "echo"
	countsID  = "remaining"
	counterID = "counter"

	actionAdd    = "add-todo"
	actionToggle = "toggle-todo"
	actionRemove = "remove-todo"
	actionFilter = "set-filter"
	actionCount  = "counter-changed"
)

// filterKey is the session value holding this page's filter. Each page picks
// its own, so it cannot be a field on the shared list.
type filterKey struct{}

// view is everything one render of the page needs.
type view struct {
	Config alacris.Config
	Items  []model.Item
	Filter string

	remaining int
}

func (v view) RemainingLabel() string {
	if v.remaining == 1 {
		return "1 left"
	}
	return fmt.Sprintf("%d left", v.remaining)
}

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dev := flag.Bool("dev", false, "serve the unminified alacris build")
	flag.Parse()

	log.SetFlags(0)
	app := &app{
		list: model.New("Read the README", "Write a component", "Ship it"),
		live: live.New(live.Options{Logger: slog.Default()}),
		dev:  *dev,
	}
	defer app.live.Close()

	app.routes()

	srv := &http.Server{
		Addr:              *addr,
		Handler:           app.mux,
		ReadHeaderTimeout: 10 * time.Second,
		// No WriteTimeout: an SSE stream is meant to stay open.
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		shown := *addr
		if strings.HasPrefix(shown, ":") {
			shown = "localhost" + shown
		}
		log.Printf("listening on http://%s", shown)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

type app struct {
	mux  *http.ServeMux
	list *model.List
	live *live.Server
	dev  bool
}

func (a *app) routes() {
	a.mux = http.NewServeMux()

	// The runtime, the live client and the two live endpoints.
	live.Mount(a.mux, alacris.DefaultBase, a.live)

	// The component module. In a real project this is whatever serves your
	// static assets.
	a.mux.Handle("/web/", http.FileServerFS(webFS))

	a.mux.HandleFunc("GET /{$}", a.index)

	a.handlers()
}

// index renders the page and hands it a session.
func (a *app) index(w http.ResponseWriter, r *http.Request) {
	sess := a.live.NewSession()
	sess.Set(filterKey{}, "all")

	// A reconnecting browser has missed everything sent while it was away, and
	// only the server knows what the page should look like.
	//
	// s.Context(), not r.Context(): this runs when the browser attaches, by
	// which time the request that rendered the page has finished and its
	// context has been cancelled.
	sess.OnOpen(func(s *live.Session) { a.push(s) })

	v := view{
		Config: alacris.Config{
			Dev:     a.dev,
			Modules: []string{"/web/components.js"},
			Live:    true,
			Session: sess.ID(),
		},
		Items:     a.list.Items(),
		Filter:    "all",
		remaining: a.list.Remaining(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The session id is a capability: it must not be cached by anything
	// between here and the browser it was made for.
	w.Header().Set("Cache-Control", "no-store, private")

	if err := page(v).Render(r.Context(), w); err != nil {
		slog.Error("rendering the page", "error", err)
	}
}

func (a *app) handlers() {
	srv := a.live

	live.On(srv, actionAdd, func(c *live.Ctx, d ui.TodoListAddDetail) error {
		if _, err := a.list.Add(d.Text); err != nil {
			// Nothing was added, so nothing changed; the input has already
			// cleared itself, which is the only feedback this warrants.
			return nil
		}
		a.pushAll()
		return nil
	})

	live.On(srv, actionToggle, func(c *live.Ctx, d ui.TodoListToggleDetail) error {
		a.list.Toggle(d.ID)
		a.pushAll()
		return nil
	})

	live.On(srv, actionRemove, func(c *live.Ctx, d ui.TodoListRemoveDetail) error {
		a.list.Remove(d.ID)
		a.pushAll()
		return nil
	})

	// The filter is this page's business, so it changes one session and
	// nobody else's.
	live.On(srv, actionFilter, func(c *live.Ctx, d ui.TodoListFilterDetail) error {
		switch d.Filter {
		case "all", "active", "done":
		default:
			return fmt.Errorf("unknown filter %q", d.Filter)
		}
		c.Session.Set(filterKey{}, d.Filter)
		c.Session.Element(todosID).Set("filter", d.Filter)
		return nil
	})

	// The counter owns its own state; this is only an echo of what it
	// reported, rendered by the server into a slot.
	live.On(srv, actionCount, func(c *live.Ctx, d ui.CounterChangeDetail) error {
		return c.Session.Element(echoID).
			SetHTML(c.Session.Context(), ui.ChipSlotDefault, chipText(fmt.Sprint(d.Value)))
	})
}

// push brings one page up to date.
//
// The context comes from the session, not from whatever request triggered the
// change: a page is updated because the list changed, and that has nothing to
// do with the lifetime of the request that changed it. Using the caller's
// context here would make one user's cancelled request abandon another user's
// update half-done.
func (a *app) push(s *live.Session) {
	ctx := s.Context()
	items := a.list.Items()
	filter, _ := s.Get(filterKey{})

	// One frame, so the list and the count land together rather than in two
	// paints.
	s.Batch(func() {
		s.Element(todosID).Set("items", items)
		if f, ok := filter.(string); ok {
			s.Element(todosID).Set("filter", f)
		}
		if err := s.Element(countsID).SetHTML(ctx, ui.ChipSlotDefault,
			chipText(view{remaining: a.list.Remaining()}.RemainingLabel())); err != nil {
			slog.Error("rendering the remaining count", "error", err)
		}
	})
}

// pushAll brings every open page up to date, which is what makes two tabs
// agree without either of them polling.
func (a *app) pushAll() {
	for _, s := range a.live.Sessions() {
		a.push(s)
	}
}
