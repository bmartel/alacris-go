// Package live makes the server authoritative over component state.
//
// The idea is small. An alacris prop is a signal and a DOM property, so a
// change on the server can be expressed as "set this property on this element"
// — one write, one binding, one DOM node. There is no HTML on the wire after
// first paint, nothing to diff, nothing to morph, and nothing that disturbs
// focus, scroll position or what the user has typed.
//
// Down, over Server-Sent Events:
//
//	sess.Element("cart").Set("count", 3)
//
// Up, over an ordinary POST: a component's CustomEvents are forwarded to named
// server actions by rendering an element with On.
//
//	@ui.TodoList(props).ID("todos").On(ui.TodoListEventAdd, "add-todo")
//
//	live.On(srv, "add-todo", func(c *live.Ctx, d ui.TodoListAddDetail) error {
//	    list.Add(d.Text)
//	    c.Session.Element("todos").Set("items", list.Items())
//	    return nil
//	})
//
// This costs a stateful server and session affinity behind a load balancer.
// The rest of this module does not depend on it: rendering components and
// generating wrappers are ordinary request/response work.
//
// # Security
//
// A session id is a capability. It is generated with crypto/rand and it must
// only ever reach the page it was created for — not a URL that could be
// shared, logged or sent as a referer. Action payloads are input like any
// other: bound to a Go type, size-limited, and worth validating.
package live

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Defaults for Options.
const (
	DefaultTTL       = 5 * time.Minute
	DefaultBuffer    = 256
	DefaultMaxDetail = 64 << 10
	DefaultHeartbeat = 25 * time.Second
)

// Options configure a Server.
type Options struct {
	// TTL is how long a session survives with no browser attached, covering
	// reloads and flaky connections. Defaults to DefaultTTL.
	TTL time.Duration

	// Buffer is how many patches are held for a session that has no browser
	// attached. Past it the oldest are dropped. Defaults to DefaultBuffer.
	Buffer int

	// MaxDetail is the largest action payload accepted, in bytes.
	// Defaults to DefaultMaxDetail.
	MaxDetail int64

	// Heartbeat is how often a comment is written to an idle stream, to stop
	// proxies from closing it. Defaults to DefaultHeartbeat.
	Heartbeat time.Duration

	// AllowOrigin reports whether an action may be accepted from this Origin.
	// When nil, only same-origin requests and requests with no Origin header
	// are accepted.
	AllowOrigin func(origin string, r *http.Request) bool

	// Logger receives handler-level problems. Defaults to slog.Default().
	Logger *slog.Logger

	// Now overrides the clock, for tests.
	Now func() time.Time
}

func (o *Options) fill() {
	if o.TTL <= 0 {
		o.TTL = DefaultTTL
	}
	if o.Buffer <= 0 {
		o.Buffer = DefaultBuffer
	}
	if o.MaxDetail <= 0 {
		o.MaxDetail = DefaultMaxDetail
	}
	if o.Heartbeat <= 0 {
		o.Heartbeat = DefaultHeartbeat
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Now == nil {
		o.Now = time.Now
	}
}

// A Server holds the live sessions and serves their transport.
//
// It implements http.Handler and, like the runtime handler, resolves requests
// by the last part of the path, so it works at any mount point:
//
//	mux.Handle("/_alacris/", live.New())
type Server struct {
	opts Options

	mu       sync.RWMutex
	sessions map[string]*Session
	actions  map[string]Handler

	stop     chan struct{}
	stopOnce sync.Once
}

var _ http.Handler = (*Server)(nil)

// New returns a running Server. Close it when the process is done with it.
func New(opts ...Options) *Server {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	o.fill()

	s := &Server{
		opts:     o,
		sessions: map[string]*Session{},
		actions:  map[string]Handler{},
		stop:     make(chan struct{}),
	}
	go s.collect()
	return s
}

func (s *Server) now() time.Time { return s.opts.Now() }

// NewSession creates a session for one page render.
func (s *Server) NewSession() *Session {
	sess := &Session{
		id:       newSessionID(),
		srv:      s,
		lastSeen: s.now(),
		actions:  map[string]Handler{},
	}
	s.mu.Lock()
	s.sessions[sess.id] = sess
	s.mu.Unlock()
	return sess
}

// Session looks up a session by id.
func (s *Server) Session(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

// Sessions returns every live session, for broadcasting.
func (s *Server) Sessions() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		out = append(out, sess)
	}
	return out
}

// Broadcast sends patches to every session.
func (s *Server) Broadcast(patches ...Patch) {
	for _, sess := range s.Sessions() {
		sess.Send(patches...)
	}
}

func (s *Server) remove(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// Close ends every session and stops the collector.
func (s *Server) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
	for _, sess := range s.Sessions() {
		sess.Close()
	}
}

// collect drops sessions whose browser has been gone longer than the TTL.
func (s *Server) collect() {
	interval := s.opts.TTL / 2
	if interval < time.Second {
		interval = time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			now := s.now()
			for _, sess := range s.Sessions() {
				if sess.expired(now, s.opts.TTL) {
					sess.Close()
				}
			}
		}
	}
}

// ServeHTTP routes the live endpoints:
//
//	GET  .../live.js   the client script
//	GET  .../live?s=   the patch stream
//	POST .../live      an action
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch name := lastSegment(r.URL.Path); {
	case name == "live.js":
		serveClient(w, r)
	case name == "live" && r.Method == http.MethodGet:
		s.serveStream(w, r)
	case name == "live" && r.Method == http.MethodPost:
		s.serveAction(w, r)
	case name == "live":
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	default:
		http.NotFound(w, r)
	}
}

func lastSegment(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

func (s *Server) serveStream(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.Session(r.URL.Query().Get("s"))
	if !ok {
		// An expired or unknown session is not an error the page can recover
		// from by retrying, but EventSource will retry regardless; 404 keeps
		// the log honest about which it was.
		http.Error(w, "unknown session", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	patches, backlog, release, err := sess.subscribe()
	if err != nil {
		http.Error(w, "session closed", http.StatusGone)
		return
	}
	defer release()

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")
	// Tell nginx not to buffer, or nothing arrives until the stream ends.
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// Ask the browser to wait a moment before reconnecting, and get the first
	// bytes out so the connection is established.
	fmt.Fprintf(w, "retry: 2000\n\n")
	flusher.Flush()

	if len(backlog) > 0 {
		if !writeFrame(w, backlog, s.opts.Logger) {
			return
		}
		flusher.Flush()
	}

	beat := time.NewTicker(s.opts.Heartbeat)
	defer beat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return

		case <-s.stop:
			return

		case frame, open := <-patches:
			if !open {
				return
			}
			if !writeFrame(w, frame, s.opts.Logger) {
				return
			}
			flusher.Flush()

		case <-beat.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeFrame(w http.ResponseWriter, patches []Patch, log *slog.Logger) bool {
	body, err := json.Marshal(patches)
	if err != nil {
		// One unserialisable value must not kill the stream, but it does mean
		// the page is now out of date in a way it cannot detect.
		log.Error("alacris live: dropping a frame that will not encode", "error", err)
		return true
	}
	// json.Marshal escapes newlines, so the payload is always one SSE line.
	if _, err := fmt.Fprintf(w, "data: %s\n\n", body); err != nil {
		return false
	}
	return true
}
