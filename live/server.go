// Package live makes the server authoritative over component state.
//
// An alacris prop is a signal and a DOM property, so a change on the server
// is "set this property on this element": one write, one binding, one DOM
// node. There is no HTML on the wire after first paint, and nothing that
// disturbs focus, scroll position or what the user has typed.
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
// Driving a page takes two values, and neither is enough alone.
//
// The capability is a client token in an HttpOnly, SameSite, path-scoped
// cookie. Script cannot read it, it is not sent cross-site, and it never
// appears in a page, a URL or a log.
//
// The page id says which of a browser's open pages is talking. It travels in
// the stream's query string, because EventSource cannot set headers, so it
// does reach access logs. On its own it grants nothing.
// There is nothing here to scrub.
//
// See cookie.go for why the two are split rather than combined.
//
// Action payloads are input like any other: bound to a Go type, size-limited.
// Validate the values.
//
// Registering an action makes it callable: any browser holding a valid
// session can invoke any registered action, with any element id and any
// detail, regardless of what the page rendered. Authorisation is the
// handler's job: check the session's own state (who this page belongs to,
// what it may touch) inside the handler, not the wiring on the page.
package live

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	alacris "github.com/bmartel/alacris-go"
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

	// CookieName is the cookie the client token is stored in.
	// Defaults to DefaultCookieName.
	CookieName string

	// CookiePath scopes the cookie so it is not sent with every request to the
	// rest of the application. It has to cover the live endpoint. Defaults to
	// alacris.DefaultBase; Mount sets it to match the base it is given.
	CookiePath string

	// CookieDomain is left empty for a host-only cookie, which is what you
	// want unless the page and the live endpoint are on different subdomains.
	CookieDomain string

	// CookieSecure decides whether the cookie carries Secure. Defaults to
	// SecureAuto, which sets it for requests that arrived over TLS.
	CookieSecure Secure

	// CookieSameSite defaults to http.SameSiteLaxMode, which is what stops a
	// cross-site page from making the browser attach the cookie to a request
	// it forged. Only weaken it to None when the page and the live endpoint
	// are genuinely on different origins, and then only with Secure and an
	// AllowOrigin that names the origins you trust.
	CookieSameSite http.SameSite

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
	if o.CookieName == "" {
		o.CookieName = DefaultCookieName
	}
	if o.CookiePath == "" {
		o.CookiePath = alacris.DefaultBase
	}
	// Not http.SameSiteDefaultMode: that constant is 1, not the zero value, and
	// it means "send no SameSite attribute at all" — which leaves the decision
	// to whatever the browser defaults to today. The attribute is doing real
	// work here, so it is always written.
	if o.CookieSameSite == 0 {
		o.CookieSameSite = http.SameSiteLaxMode
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

// mustBeInitialized turns the nil-map panic a zero-value Server would produce
// into one that names the fix.
func (s *Server) mustBeInitialized() {
	if s.stop == nil {
		panic("live: a Server must be created with live.New, not a struct literal — a zero Server has no collector and no session table")
	}
}

// NewSession creates a session for one page render, and gives the browser the
// cookie that will authorise it.
//
// It needs the exchange because the session is bound to a client token, and
// that token is a cookie: read from the request when the browser already has
// one, minted and set on the response when it does not. Call it before
// anything is written to w — a cookie cannot be set once the headers are out.
//
// The session's context is derived from context.Background rather than from
// the request: it has to outlive it, because OnOpen and every action handler
// run long after the render has finished.
func (s *Server) NewSession(w http.ResponseWriter, r *http.Request) *Session {
	s.mustBeInitialized()
	ctx, cancel := context.WithCancel(context.Background())
	sess := &Session{
		id:       newToken(),
		client:   s.issueToken(w, r),
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

// evictionBatch is how far under the cap eviction aims, as a fraction of
// MaxSessions. Evicting one session per render at the cap costs an O(n log n)
// rank of the whole table on every render — 1.5ms per page at the default
// 10,000, which turns the backstop into an amplifier for exactly the overload
// it exists to absorb. Retiring the ~1% longest-idle disconnected sessions in
// one pass amortises that rank across the next ~1% of renders; the extras are
// the ones the very next renders would have evicted anyway, and connected
// sessions are never part of the surplus.
const evictionBatch = 100

// evict closes the least useful sessions to get back under the cap, plus a
// margin of the longest-idle disconnected ones so the rank is not repaid on
// the very next render (see evictionBatch).
//
// A session with a browser attached is someone looking at a page, so idle ones
// go first and, among those, the ones idle longest. Only the first n closes —
// the ones the cap strictly requires — may take a connected session; the
// batch margin never does.
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

	want := n
	if batch := s.opts.MaxSessions / evictionBatch; batch > want {
		want = batch
	}
	closed := 0
	for _, r := range ranked {
		if closed >= want || (closed >= n && r.connected) {
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

// setCookiePath scopes the cookie to where the endpoints are mounted. Mount
// calls it; it is only meaningful before anything is served.
func (s *Server) setCookiePath(base string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opts.CookiePath = base
}

// cookiePath reads the path under the same lock setCookiePath writes it, so a
// Mount racing an early NewSession is a defined read rather than a data race.
func (s *Server) cookiePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.opts.CookiePath
}

// Session looks up a session by its page id.
//
// This is the unauthorised lookup, for server-side code that already knows
// which page it means. Anything acting on a request has to go through
// sessionFor, which also checks that the request's cookie owns the page.
func (s *Server) Session(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

// sessionFor resolves the page a request is talking about, and reports whether
// the request is allowed to.
//
// Both halves are required: the cookie proves which browser, the page id says
// which of its pages. A page id on its own — out of an access log, say — names
// a session it cannot reach.
func (s *Server) sessionFor(r *http.Request, pageID string) (*Session, bool) {
	token, ok := s.clientToken(r)
	if !ok || pageID == "" {
		return nil, false
	}
	sess, ok := s.Session(pageID)
	if !ok || !sameToken(sess.client, token) {
		return nil, false
	}
	return sess, true
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
//	GET  .../live?p=   the patch stream, for the page id in p
//	POST .../live      an action
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch name := lastSegment(r.URL.Path); {
	case name == "live.js":
		serveClient(w, r)
	case name == "live" && r.Method == http.MethodOptions:
		s.servePreflight(w, r)
	case name == "live" && r.Method == http.MethodGet:
		s.serveStream(w, r)
	case name == "live" && r.Method == http.MethodPost:
		s.serveAction(w, r)
	case name == "live":
		w.Header().Set("Allow", "GET, POST, OPTIONS")
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
	// The cookie is not sent cross-site under SameSite, so this normally fails
	// closed for a hostile page before the origin check is reached. The check
	// is still here because SameSite is a browser behaviour and this is an
	// authorisation decision.
	if !s.originAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	s.allowCORS(w, r)

	sess, ok := s.sessionFor(r, r.URL.Query().Get("p"))
	if !ok {
		// Unknown page, expired session, or a page this browser does not own.
		// They are answered the same way on purpose: distinguishing them would
		// turn this endpoint into an oracle for which page ids exist.
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
	addVary(h, "Cookie")
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
			// Coalesce whatever else has already queued into this frame. A
			// handler that makes several Set calls delivers several channel
			// sends; merging them here turns that burst into one encode, one
			// write and one flush, without changing what a handler that
			// genuinely streams (patch, wait, patch) observes — an empty
			// channel is left alone.
			closed := false
			for !closed {
				select {
				case more, ok := <-patches:
					if !ok {
						closed = true
						break
					}
					frame = append(frame, more...)
					continue
				default:
				}
				break
			}
			if !writeFrame(w, frame, s.opts.Logger) {
				return
			}
			flusher.Flush()
			if closed {
				return
			}

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
	// Three plain writes land in net/http's own buffer; fmt's verb machinery
	// bought nothing on the hottest write path.
	if _, err := io.WriteString(w, "data: "); err != nil {
		return false
	}
	if _, err := w.Write(body); err != nil {
		return false
	}
	if _, err := io.WriteString(w, "\n\n"); err != nil {
		return false
	}
	return true
}
