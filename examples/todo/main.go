// Command todo is the alacris-go example — a live board.
//
//	go run ./examples/todo
//	open http://localhost:8080
//
//	go run ./examples/todo -demo   # a collaborator moves cards, for filming
//
// The board lives in Go. Every click is one property write — no HTML on the
// wire — so a card that moves in another window is the same node here, and
// whatever you were typing stays put.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
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

const (
	boardID   = "board"
	membersID = "members"

	actionAdd        = "add-card"
	actionMove       = "move-card"
	actionAddList    = "add-list"
	actionMoveList   = "move-list"
	actionRenameList = "rename-list"
	actionDeleteList = "delete-list"
	actionEdit       = "edit-card"
	actionRemove     = "remove-card"
)

// view is everything one render of the page needs.
type view struct {
	Config  alacris.Config
	Items   []model.Item
	Columns []model.Column
	Members []string
}

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dev := flag.Bool("dev", false, "serve the unminified alacris build")
	demo := flag.Bool("demo", false, "a collaborator moves cards, so one window is enough to film")
	flag.Parse()

	log.SetFlags(0)
	app := &app{
		list: model.New(),
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

	if *demo {
		go app.collaborate(ctx)
	}

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

	live.Mount(a.mux, alacris.DefaultBase, a.live)
	a.mux.Handle("/web/", http.FileServerFS(webFS))
	a.mux.HandleFunc("GET /{$}", a.index)
	a.handlers()
}

func (a *app) index(w http.ResponseWriter, r *http.Request) {
	sess := a.live.NewSession(w, r)

	// A reconnecting browser has missed everything sent while it was away, and
	// only the server knows what the page should look like.
	//
	// s.Context(), not r.Context(): this runs when the browser attaches, by
	// which time the request that rendered the page has finished and its
	// context has been cancelled.
	sess.OnOpen(func(s *live.Session) { a.pushAll() })

	v := view{
		Config: alacris.Config{
			Dev:     a.dev,
			UI:      true,
			Modules: []string{"/web/components.js"},
			Live:    true,
			Page:    sess.ID(),
		},
		Items:   a.list.Items(),
		Columns: a.list.Columns(),
		Members: a.list.Members(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, private")

	if err := page(v).Render(r.Context(), w); err != nil {
		slog.Error("rendering the page", "error", err)
	}
}

func (a *app) handlers() {
	srv := a.live

	live.On(srv, actionAdd, func(c *live.Ctx, d ui.BoardAddDetail) error {
		if _, err := a.list.Add(d.Text, d.Column); err != nil {
			return nil
		}
		a.pushAll()
		return nil
	})

	live.On(srv, actionMove, func(c *live.Ctx, d ui.BoardMoveDetail) error {
		a.list.Move(d.ID, d.Column, d.Index)
		a.pushAll()
		return nil
	})

	live.On(srv, actionAddList, func(c *live.Ctx, d ui.BoardAddlistDetail) error {
		if _, err := a.list.AddColumn(d.Title); err != nil {
			return nil
		}
		a.pushAll()
		return nil
	})

	live.On(srv, actionMoveList, func(c *live.Ctx, d ui.BoardMovelistDetail) error {
		a.list.MoveColumn(d.ID, d.Index)
		a.pushAll()
		return nil
	})

	live.On(srv, actionRenameList, func(c *live.Ctx, d ui.BoardRenameDetail) error {
		a.list.RenameColumn(d.ID, d.Title)
		a.pushAll()
		return nil
	})

	live.On(srv, actionDeleteList, func(c *live.Ctx, d ui.BoardDeletelistDetail) error {
		a.list.RemoveColumn(d.ID)
		a.pushAll()
		return nil
	})

	live.On(srv, actionEdit, func(c *live.Ctx, d ui.BoardEditDetail) error {
		a.list.Update(d.ID, d.Text, d.Body, d.Who, d.Labels)
		a.pushAll()
		return nil
	})

	live.On(srv, actionRemove, func(c *live.Ctx, d ui.BoardRemoveDetail) error {
		a.list.Remove(d.ID)
		a.pushAll()
		return nil
	})
}

// push brings one page up to date.
//
// The context comes from the session, not from whatever request triggered the
// change: a page is updated because the board changed, and that has nothing to
// do with the lifetime of the request that changed it.
func (a *app) push(s *live.Session) {
	ctx := s.Context()
	items := a.list.Items()

	s.Batch(func() {
		h := ui.BoardElement(s, boardID)
		h.SetItems(items)
		h.SetColumns(a.list.Columns())
		if err := s.Element(membersID).SetHTML(ctx, "",
			facepile(a.list.Members(), len(items))); err != nil {
			slog.Error("rendering the members", "error", err)
		}
	})
}

func (a *app) pushAll() {
	for _, s := range a.live.Sessions() {
		a.push(s)
	}
}

// collaborate is the other person in a one-window recording: it advances a
// card every few seconds so the board is visibly live while you type.
func (a *app) collaborate(ctx context.Context) {
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	n := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			ids := a.list.IDs()
			if len(ids) == 0 {
				continue
			}
			cols := a.list.ColumnIDs()
			if len(cols) == 0 {
				continue
			}
			a.list.Move(ids[n%len(ids)], cols[n%len(cols)], 0)
			n++
			a.pushAll()
		}
	}
}
