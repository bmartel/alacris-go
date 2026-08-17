package live

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/a-h/templ"
)

// ErrClosed is returned when a session has been closed or has expired.
var ErrClosed = errors.New("live: session is closed")

// A Session is one page's connection to the server.
//
// It is identified by two values. Its ID names the page and is safe to put in
// the page and in the stream URL. The capability is the client token in the
// browser's cookie, which never appears in either. A request needs both, so a
// page id recovered from a log or a referer reaches nothing on its own.
type Session struct {
	// id identifies the page. It travels in the stream URL, so it is an
	// identifier rather than a secret.
	id string

	// client is the token from the browser's cookie. It is the capability, and
	// a request has to present it to reach this session.
	client string

	srv *Server

	// ctx lives as long as the session, which is what work started from a
	// callback needs — the request that created the session is long gone by
	// the time OnOpen or an action handler runs.
	ctx    context.Context
	cancel context.CancelFunc

	mu        sync.Mutex
	pending   []Patch
	batching  int
	batch     []Patch
	sub       chan []Patch
	lastSeen  time.Time
	connected bool
	closed    bool
	dropped   bool

	actions map[string]Handler
	onOpen  func(*Session)
	values  map[any]any
}

// ID returns the page id, which is what goes in the page as alacris.Config.Page.
//
// It is not a secret. The secret is the client token in the cookie, which this
// package sets and the browser sends; it is never exposed here, because
// nothing outside the package has any use for it.
func (s *Session) ID() string { return s.id }

// Context returns a context that lives as long as the session and is cancelled
// when it closes.
//
// This is the context to pass to SetHTML, or to anything else started from
// OnOpen or an action handler that outlives the request it was triggered by.
// Reaching for the request's context instead is the mistake this exists to
// prevent: the request that rendered the page has finished by the time OnOpen
// runs, so its context is already cancelled and the render fails.
func (s *Session) Context() context.Context { return s.ctx }

// Closed reports whether the session has been closed or has expired.
func (s *Session) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Connected reports whether a browser is currently attached.
func (s *Session) Connected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected
}

// Set stores a value on the session. It is where per-page server state goes
// when there is nowhere better for it.
func (s *Session) Set(key, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = map[any]any{}
	}
	s.values[key] = value
}

// Get reads a value stored with Set.
func (s *Session) Get(key any) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	return v, ok
}

// OnOpen registers a function to run whenever a browser attaches, including
// after a reconnect.
//
// Register one. A reconnecting EventSource has missed everything sent while it
// was away, and the server is the only side that knows what the page should
// look like — so push the full state here rather than assuming the page kept
// up.
func (s *Session) OnOpen(fn func(*Session)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onOpen = fn
}

// Element returns a handle for patching one element by its id.
func (s *Session) Element(id string) Handle { return Handle{s: s, id: id} }

// A Handle addresses one element in one session.
type Handle struct {
	s  *Session
	id string
}

// ID returns the element id this handle targets.
func (h Handle) ID() string { return h.id }

// Set writes a component prop. name is the JavaScript prop name from define().
func (h Handle) Set(name string, value any) { h.s.Send(Prop(h.id, name, value)) }

// SetAttr writes an ordinary attribute; a nil value removes it.
func (h Handle) SetAttr(name string, value any) { h.s.Send(Attr(h.id, name, value)) }

// Class turns a class name on or off.
func (h Handle) Class(name string, on bool) { h.s.Send(Class(h.id, name, on)) }

// SetHTML replaces the children assigned to one slot with rendered markup.
func (h Handle) SetHTML(ctx context.Context, slot string, c templ.Component) error {
	p, err := HTML(ctx, h.id, slot, c)
	if err != nil {
		return err
	}
	h.s.Send(p)
	return nil
}

// Props writes several props to the same element as one frame.
func (h Handle) Props(props map[string]any) {
	h.s.Batch(func() {
		for name, value := range props {
			h.Set(name, value)
		}
	})
}

// Send queues patches for the browser.
//
// It never blocks and never fails: a session with no browser attached buffers,
// and a closed one discards. A page that is not there cannot be updated, and
// making every call site handle that would put error checks around code whose
// only correct response is to carry on.
func (s *Session) Send(patches ...Patch) {
	patches = s.vetted(patches)
	if len(patches) == 0 {
		return
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	if s.batching > 0 {
		s.batch = append(s.batch, patches...)
		s.mu.Unlock()
		return
	}
	ok := s.deliver(patches)
	s.mu.Unlock()

	if !ok {
		s.resync()
	}
}

// vetted drops any patch whose key would let a state write execute script —
// see unsafePatch. The common case is that nothing is dropped, and it costs
// one scan and no allocation.
func (s *Session) vetted(patches []Patch) []Patch {
	bad := -1
	for i := range patches {
		if unsafePatch(patches[i]) != "" {
			bad = i
			break
		}
	}
	if bad < 0 {
		return patches
	}
	out := make([]Patch, 0, len(patches)-1)
	for i := range patches {
		if reason := unsafePatch(patches[i]); reason != "" {
			s.srv.opts.Logger.Error("alacris live: dropping a patch that could execute script",
				"session", s.id, "op", string(patches[i].Op), "key", patches[i].Key, "reason", reason)
			continue
		}
		out = append(out, patches[i])
	}
	return out
}

// deliver buffers patches when nothing is attached, otherwise hands them to
// the subscriber. It reports whether the frame was taken; false means the
// subscriber has stopped keeping up and the caller should resync.
//
// It must be called with s.mu held, and the send must stay under the lock.
// Every path that closes s.sub — subscribe, release, resync, Close — first
// nils the field under this same lock, so a send made under the lock can
// never see a closed channel. A send made outside the lock on a channel
// captured earlier can, and a select send on a closed channel panics rather
// than taking default. The lock cannot stall the caller on a slow reader,
// because the send is non-blocking either way.
func (s *Session) deliver(patches []Patch) bool {
	if s.sub == nil {
		s.pending = append(s.pending, patches...)
		if max := s.srv.opts.Buffer; len(s.pending) > max {
			// Keep the newest: for a stream of prop writes, the last value of
			// each prop is the one that matters. dropped tells the reconnect
			// path that the page cannot be trusted to be up to date.
			s.pending = append(s.pending[:0], s.pending[len(s.pending)-max:]...)
			s.dropped = true
		}
		return true
	}
	// The capacity clamp keeps the frame's backing array private. Broadcast
	// hands the same slice to every session, and the stream loop coalesces
	// with append — without the clamp, an append into spare capacity would
	// write into an array other goroutines are marshalling concurrently.
	select {
	case s.sub <- patches[:len(patches):len(patches)]:
		return true
	default:
		return false
	}
}

// resync ends the stream so the browser reconnects.
//
// It is what happens when a subscriber stops keeping up. Blocking would stall
// the application on one slow reader; dropping the frame would leave the page
// wrong with nobody aware of it, because a patch is a whole value and there is
// no guarantee anything writes that prop again. Ending the stream costs a
// reconnect and gets a page that is right: EventSource comes back on its own,
// and OnOpen restates everything.
func (s *Session) resync() {
	s.mu.Lock()
	sub := s.sub
	// Clearing it under the lock means exactly one caller sees a live channel
	// here, so exactly one closes it.
	s.sub = nil
	if sub != nil {
		s.connected = false
		s.lastSeen = s.srv.now()
	}
	hasOpen := s.onOpen != nil
	s.mu.Unlock()

	if sub == nil {
		return
	}
	close(sub)
	if hasOpen {
		s.srv.opts.Logger.Warn("alacris live: subscriber fell behind, ending the stream so it reconnects",
			"session", s.id)
	} else {
		// The dropped frame is gone and nothing will restate it — this is the
		// silently-stale-page bug, caught at the moment it happens.
		s.srv.opts.Logger.Warn("alacris live: subscriber fell behind and no OnOpen is registered to restate state on reconnect; the page may now be stale",
			"session", s.id)
	}
}

// Batch coalesces every patch made inside fn into one frame, so the browser
// applies them together.
func (s *Session) Batch(fn func()) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.batching++
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.batching--
		if s.batching > 0 || len(s.batch) == 0 {
			s.mu.Unlock()
			return
		}
		batch := s.batch
		s.batch = nil
		ok := s.deliver(batch)
		s.mu.Unlock()
		if !ok {
			s.resync()
		}
	}()

	fn()
}

// Close ends the session. Patches sent afterwards are discarded.
func (s *Session) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	sub := s.sub
	s.sub = nil
	s.pending = nil
	s.mu.Unlock()

	if sub != nil {
		close(sub)
	}
	s.cancel()
	s.srv.remove(s.id)
}

// Subscribe attaches a programmatic subscriber in place of a browser: OnOpen
// runs, and every Send after that is a frame on the channel. backlog is
// whatever was buffered while nothing was attached — it precedes the channel's
// frames chronologically, which is why it is handed over rather than queued.
// Release detaches; the channel also closes when the session ends or another
// subscriber (a real browser included) replaces this one.
//
// This is the seam the livetest package records through, and it is equally
// the way to bridge patches onto a transport this package does not provide.
// It carries a browser's obligations too: a subscriber that stops draining is
// treated exactly like a slow browser — the session drops it and relies on
// OnOpen to restate state to whoever attaches next.
func (s *Session) Subscribe() (frames <-chan []Patch, backlog []Patch, release func(), err error) {
	return s.subscribe()
}

// subscribe attaches a browser. The returned channel is closed when the
// session ends or another connection replaces this one.
//
// The channel's capacity is the synchronous-delivery contract Subscribe and
// the livetest recorder rely on: Send completes its buffered channel send
// before returning, so once an action handler has returned, its frames are in
// the channel — reading them needs no waiting and no races.
func (s *Session) subscribe() (<-chan []Patch, []Patch, func(), error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, nil, nil, ErrClosed
	}

	// A reload can open the new stream before the old one has finished dying,
	// so the newest connection wins rather than being turned away.
	if old := s.sub; old != nil {
		s.sub = nil
		close(old)
	}

	ch := make(chan []Patch, 16)
	s.sub = ch
	s.connected = true
	s.lastSeen = s.srv.now()

	backlog := s.pending
	s.pending = nil
	dropped := s.dropped
	s.dropped = false
	onOpen := s.onOpen
	s.mu.Unlock()

	if onOpen != nil {
		onOpen(s)
	} else if dropped {
		s.srv.opts.Logger.Warn("alacris live: patches were dropped while no browser was attached and no OnOpen is registered to restate them; the page may be stale",
			"session", s.id)
	}

	release := func() {
		s.mu.Lock()
		if s.sub == ch {
			s.sub = nil
			s.connected = false
			s.lastSeen = s.srv.now()
			close(ch)
		}
		s.mu.Unlock()
	}
	return ch, backlog, release, nil
}

// activity reports whether a browser is attached and, if not, when the last
// one left. It is what eviction ranks on.
func (s *Session) activity() (connected bool, since time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected, s.lastSeen
}

// expired reports whether a disconnected session has outlived its grace period.
func (s *Session) expired(now time.Time, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return true
	}
	if s.connected {
		return false
	}
	return now.Sub(s.lastSeen) > ttl
}
