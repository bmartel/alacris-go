package live

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/a-h/templ"
)

// ErrClosed is returned when a session has been closed or has expired.
var ErrClosed = errors.New("live: session is closed")

// A Session is one page's connection to the server.
//
// Its ID is a capability: anyone who has it can receive that page's patches
// and send actions as that page. It is generated with crypto/rand, and it
// belongs in the page it was made for and nowhere else — not in a URL that
// might be shared, logged or referred.
type Session struct {
	id  string
	srv *Server

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

func newSessionID() string {
	var b [18]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail on any supported platform, and carrying on
		// with a guessable session id would hand one page's state to whoever
		// guesses it.
		panic("live: no randomness available: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

// ID returns the session identifier to put in the page.
func (s *Session) ID() string { return s.id }

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
	sub, out := s.take(patches)
	s.mu.Unlock()

	deliver(sub, out)
}

// take decides, under the lock, whether patches go out now or wait.
func (s *Session) take(patches []Patch) (chan []Patch, []Patch) {
	if s.sub == nil {
		s.pending = append(s.pending, patches...)
		if max := s.srv.opts.Buffer; len(s.pending) > max {
			// Keep the newest: for a stream of prop writes, the last value of
			// each prop is the one that matters. dropped tells the reconnect
			// path that the page cannot be trusted to be up to date.
			s.pending = append(s.pending[:0], s.pending[len(s.pending)-max:]...)
			s.dropped = true
		}
		return nil, nil
	}
	return s.sub, patches
}

// deliver sends outside the lock so a slow reader cannot block the caller.
func deliver(sub chan []Patch, out []Patch) {
	if sub == nil || len(out) == 0 {
		return
	}
	select {
	case sub <- out:
	default:
		// The subscriber is not keeping up. Dropping is the only option that
		// does not stall the application; the stream carries whole values, so
		// the next write of the same prop replaces what was lost.
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
		sub, out := s.take(batch)
		s.mu.Unlock()
		deliver(sub, out)
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
	s.srv.remove(s.id)
}

// subscribe attaches a browser. The returned channel is closed when the
// session ends or another connection replaces this one.
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
	s.dropped = false
	onOpen := s.onOpen
	s.mu.Unlock()

	if onOpen != nil {
		onOpen(s)
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
