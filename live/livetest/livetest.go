// Package livetest makes live action handlers unit-testable.
//
// Testing a handler used to mean a real HTTP server, a cookie exchange, and
// hand-parsed SSE frames, so handlers went untested. This package removes the
// transport: NewSession mints a session with a Recorder attached in place of a
// browser, Invoke runs a registered handler the way an incoming action would,
// and the Recorder holds every patch the handler produced.
//
//	srv := live.New()
//	defer srv.Close()
//	registerHandlers(srv) // the code under test
//
//	sess, rec := livetest.NewSession(t, srv)
//	if err := livetest.Invoke(t, sess, "add-todo", "todos", addDetail{Text: "milk"}); err != nil {
//	    t.Fatal(err)
//	}
//	if _, ok := rec.Last("todos", "items"); !ok {
//	    t.Error("add-todo did not patch the items prop")
//	}
//
// Everything is synchronous: a Send inside a handler has reached the recorder
// by the time Invoke returns, so assertions need no waiting and no polling.
package livetest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bmartel/alacris-go/live"
)

// NewSession mints a session on srv the way a page render would — cookie
// exchange included, on a synthetic request — and attaches a Recorder in
// place of a browser. The session is closed when the test ends.
//
// To exercise OnOpen the way a real page does, keep the real order: mint with
// RenderSession, register OnOpen, then Attach.
func NewSession(tb testing.TB, srv *live.Server) (*live.Session, *Recorder) {
	tb.Helper()
	sess := RenderSession(tb, srv)
	return sess, Attach(tb, sess)
}

// RenderSession mints a session the way the application's render handler
// would, with nothing attached yet — the state a page is in between render
// and the browser connecting. Register OnOpen on it, then Attach. The session
// is closed when the test ends.
func RenderSession(tb testing.TB, srv *live.Server) *live.Session {
	tb.Helper()
	sess := srv.NewSession(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	tb.Cleanup(sess.Close)
	return sess
}

// Attach connects a Recorder to the session in place of a browser: OnOpen
// runs, anything buffered since the render is recorded, and every later Send
// is visible to the recorder's reads. The recorder detaches when the test
// ends.
func Attach(tb testing.TB, sess *live.Session) *Recorder {
	tb.Helper()
	frames, backlog, release, err := sess.Subscribe()
	if err != nil {
		tb.Fatalf("livetest: subscribing to the session: %v", err)
	}
	rec := &Recorder{tb: tb, frames: frames}
	if len(backlog) > 0 {
		rec.taken = append(rec.taken, backlog)
	}
	tb.Cleanup(release)
	return rec
}

// Invoke runs the handler registered for action, session-scoped or
// server-wide, with detail as the event's payload — the same code path an
// incoming action takes, minus the transport. element becomes Ctx.Element,
// which is what Ctx.Handle addresses; leave it empty when the handler does
// not use it.
//
// detail may be nil, a json.RawMessage, or any value to marshal.
func Invoke(tb testing.TB, s *live.Session, action, element string, detail any) error {
	tb.Helper()
	var raw json.RawMessage
	switch d := detail.(type) {
	case nil:
	case json.RawMessage:
		raw = d
	case []byte:
		raw = d
	default:
		b, err := json.Marshal(d)
		if err != nil {
			tb.Fatalf("livetest: encoding the detail for %q: %v", action, err)
		}
		raw = b
	}
	return s.Dispatch(nil, action, element, raw)
}

// A Recorder holds the patches a session sent, standing in for the browser.
//
// Reads are synchronous with the handlers that produced the patches: whatever
// a handler sent before returning is visible to the next Patches, Frames or
// Last call. Like a browser, the recorder must keep up — a session drops a
// subscriber more than 16 frames behind, so read (or batch) between bursts.
type Recorder struct {
	tb     testing.TB
	frames <-chan []live.Patch
	taken  [][]live.Patch
	ended  bool
}

// drain moves everything the session has queued into the recorder.
func (r *Recorder) drain() {
	for {
		select {
		case f, ok := <-r.frames:
			if !ok {
				if !r.ended {
					r.ended = true
					r.tb.Logf("livetest: the session detached this recorder (closed, replaced, or fell behind)")
				}
				return
			}
			r.taken = append(r.taken, f)
		default:
			return
		}
	}
}

// Frames returns every frame so far, in delivery order. Patches batched
// together arrive as one frame, so frame boundaries are the "lands in one
// paint" contract Batch makes.
func (r *Recorder) Frames() [][]live.Patch {
	r.drain()
	out := make([][]live.Patch, len(r.taken))
	copy(out, r.taken)
	return out
}

// Patches returns every patch so far, flattened, in order.
func (r *Recorder) Patches() []live.Patch {
	r.drain()
	var out []live.Patch
	for _, f := range r.taken {
		out = append(out, f...)
	}
	return out
}

// Last returns the most recent patch for one element and key — the value the
// page would be showing — and whether any patch matched at all.
func (r *Recorder) Last(elementID, key string) (live.Patch, bool) {
	all := r.Patches()
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].ID == elementID && all[i].Key == key {
			return all[i], true
		}
	}
	return live.Patch{}, false
}

// Reset forgets everything recorded so far, so the next assertion reads only
// what the next action produced.
func (r *Recorder) Reset() {
	r.drain()
	r.taken = nil
}

// String summarises the recording, for t.Log and failure messages.
func (r *Recorder) String() string {
	r.drain()
	patches := 0
	for _, f := range r.taken {
		patches += len(f)
	}
	return fmt.Sprintf("livetest.Recorder(%d frames, %d patches)", len(r.taken), patches)
}
