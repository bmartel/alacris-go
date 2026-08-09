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
// only ever reach the page it was created for. Note that the client puts it in
// the stream URL, because EventSource cannot set headers, so it reaches your
// access logs too: scrub the "s" query parameter, or treat log access as
// equivalent to being able to drive any live page. It never appears in a
// page's own URL, so it does not travel in a Referer or a shared link.
//
// Action payloads are input like any other: bound to a Go type, size-limited,
// and worth validating.
package live

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Defaults for Options.
const (
	DefaultTTL         = 5 * time.Minute
	DefaultBuffer      = 256
	DefaultMaxDetail   = 64 << 10
	DefaultHeartbeat   = 25 * time.Second
	DefaultMaxSessions = 10_000
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

	// MaxSessions bounds how many sessions are held at once. Past it, the
	// least recently active session with no browser attached is closed to make
	// room. Defaults to DefaultMaxSessions.
	//
	// A session is usually created per page render, which for most
	// applications means per unauthenticated GET. Without a bound, anything
	// that follows links — a crawler, a scanner, a load test — leaves a
	// session behind for every request, each holding a buffer, none of them
	// expiring until their TTL. This is a backstop, not a substitute for rate
	// limiting the handler that creates them.
	MaxSessions int

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
	if o.MaxSessions <= 0 {
		o.MaxSessions = DefaultMaxSessions
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
	closed   bool

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
//
// The session's context is derived from context.Background rather than from
// the request that created it: it has to outlive that request, because OnOpen
// and every action handler run long after it has finished.
func (s *Server) NewSession() *Session {
	ctx, cancel := context.WithCancel(context.Background())
	sess := &Session{
		id:       newSessionID(),
		srv:      s,
		ctx:      ctx,
		cancel:   cancel,
		lastSeen: s.now(),
		actions:  map[string]Handler{},
	}
	s.mu.Lock()
	if s.closed {
		// The collector has stopped, so a session registered now would never
		// expire. Hand back a closed one instead: Send is a no-op on it, which
		// is what a caller racing shutdown wants anyway.
		s.mu.Unlock()
		cancel()
		sess.closed = true
		return sess
	}
	s.sessions[sess.id] = sess
	over := len(s.sessions) - s.opts.MaxSessions
	var others []*Session
	if over > 0 {
		others = make([]*Session, 0, len(s.sessions)-1)
		for _, other := range s.sessions {
			if other != sess {
				others = append(others, other)
			}
		}
	}
	s.mu.Unlock()

	// Deliberately outside the lock. Deciding what to evict means reading each
	// session's own state, and every other path in this package takes a
	// session's lock before the server's — taking them the other way round
	// here would be the one inversion that deadlocks.
	if over > 0 {
		s.evict(others, over)
	}
	return sess
}

// evict closes the least useful sessions to get back under the cap.
//
// A session with a browser attached is someone looking at a page, so idle ones
// go first and, among those, the ones idle longest.
func (s *Server) evict(candidates []*Session, n int) {
	type scored struct {
		sess      *Session
		connected bool
		idleSince time.Time
	}

	ranked := make([]scored, 0, len(candidates))
	for _, sess := range candidates {
		connected, since := sess.activity()
		ranked = append(ranked, scored{sess: sess, connected: connected, idleSince: since})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].connected != ranked[j].connected {
			return !ranked[i].connected
		}
		return ranked[i].idleSince.Before(ranked[j].idleSince)
	})

	closed := 0
	for _, r := range ranked {
		if closed >= n {
			break
		}
		r.sess.Close()
		closed++
	}
	if closed > 0 {
		s.opts.Logger.Warn("alacris live: session limit reached, closing the least active",
			"closed", closed, "limit", s.opts.MaxSessions)
	}
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
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
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
