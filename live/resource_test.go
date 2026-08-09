package live

import (
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"
)

// A subscriber that stops reading used to have its frames dropped on the
// floor. The comment claimed the next write of the same prop would repair it,
// which is only true if there is a next write: a one-off patch — the list after
// a delete, say — was lost, and the page stayed wrong for as long as it was
// open, with nothing on either side aware of it.
//
// Ending the stream instead turns silent corruption into a reconnect, and the
// reconnect runs OnOpen, which restates everything.
func TestASlowSubscriberIsResyncedRatherThanSilentlyDropped(t *testing.T) {
	srv := New(Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	defer srv.Close()

	sess := newSession(srv)

	// Subscribe, then never read: the channel fills and stays full.
	patches, _, release, err := sess.subscribe()
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	for i := 0; i < 1000; i++ {
		sess.Element("x").Set("n", i)
		if !sess.Connected() {
			break
		}
	}

	if sess.Connected() {
		t.Fatal("the session is still attached after overflowing the subscriber; frames are being dropped silently")
	}

	// The stream has to actually end, or the browser never reconnects and
	// never gets a fresh state.
	deadline := time.After(3 * time.Second)
	for {
		select {
		case _, open := <-patches:
			if !open {
				goto closed
			}
		case <-deadline:
			t.Fatal("the stream was never closed, so the browser would not reconnect")
		}
	}
closed:

	// Everything after the resync buffers for the next connection, so nothing
	// sent in the gap is lost.
	sess.Element("x").Set("n", "after")
	sess.mu.Lock()
	pending := len(sess.pending)
	sess.mu.Unlock()
	if pending == 0 {
		t.Error("patches sent after a resync should buffer for the reconnecting browser")
	}
}

// Sessions are created per page render, which for most applications means per
// unauthenticated GET. Without a bound, a few minutes of traffic from anything
// that follows links is a few hundred thousand live sessions, each holding a
// buffer, none of them expiring until their TTL.
func TestSessionsAreBounded(t *testing.T) {
	const max = 32

	srv := New(Options{
		MaxSessions: max,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	defer srv.Close()

	var ids []string
	for i := 0; i < max*10; i++ {
		ids = append(ids, newSession(srv).ID())
	}

	if got := len(srv.Sessions()); got > max {
		t.Errorf("%d sessions are live with MaxSessions=%d", got, max)
	}

	// The most recent ones are the ones worth keeping.
	newest := ids[len(ids)-1]
	if _, ok := srv.Session(newest); !ok {
		t.Error("the newest session was evicted")
	}
	oldest := ids[0]
	if _, ok := srv.Session(oldest); ok {
		t.Error("the oldest session survived the cap")
	}
}

// An attached browser is a real user. A disconnected session is a tab that may
// never come back, so it is the one to give up first.
func TestEvictionPrefersDisconnectedSessions(t *testing.T) {
	const max = 4

	srv, ts := newTestServer(t, Options{MaxSessions: max})

	attached := newSession(srv)
	frames, stop := stream(t, ts, attached)
	defer stop()
	waitConnected(t, attached)
	_ = frames

	for i := 0; i < max*4; i++ {
		newSession(srv)
	}

	if _, ok := srv.Session(attached.ID()); !ok {
		t.Error("a session with a browser attached was evicted before the idle ones")
	}
}

// A session created after the server shut down would never be collected: the
// sweeper is gone. Handing back a closed one keeps a caller racing shutdown
// from quietly accumulating sessions that outlive the process's intent.
func TestNewSessionAfterCloseIsInert(t *testing.T) {
	srv := New(Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	srv.Close()

	sess := newSession(srv)
	if !sess.Closed() {
		t.Error("a session created after Close should already be closed")
	}
	if _, ok := srv.Session(sess.ID()); ok {
		t.Error("a session created after Close should not be retained by the server")
	}
	// Still safe to use, because callers should not have to check.
	sess.Element("x").Set("n", 1)
	sess.Close()
}

// Every Server starts a sweeper. Creating and closing them in a loop — which a
// test suite or a per-tenant setup does — must not leave those behind.
func TestClosingAServerLeavesNoGoroutines(t *testing.T) {
	quiet := Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), TTL: time.Second}

	// Settle whatever the runtime is already doing.
	before := goroutinesAfterSettling()

	for i := 0; i < 50; i++ {
		srv := New(quiet)
		newSession(srv)
		newSession(srv)
		srv.Close()
	}

	if after := goroutinesAfterSettling(); after > before+5 {
		t.Errorf("goroutines went from %d to %d across 50 server lifetimes", before, after)
	}
}

func goroutinesAfterSettling() int {
	var n int
	for i := 0; i < 50; i++ {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		if got := runtime.NumGoroutine(); got == n {
			return n
		} else {
			n = got
		}
	}
	return n
}

// The origin check is the only thing standing between a hostile page and an
// action, once it has somehow learned a session id. The rejection case was
// already covered; this pins the two acceptances so a stricter rule cannot
// silently break every real request.
func TestOriginRules(t *testing.T) {
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	ran := make(chan string, 8)
	srv.On("act", func(c *Ctx) error {
		ran <- c.Action
		return nil
	})
	body := `{"p":"` + sess.ID() + `","a":"act","i":"","d":null}`

	send := func(t *testing.T, origin string, want int) {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/_alacris/live", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(sessionCookie(sess))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
		if resp.StatusCode != want {
			t.Errorf("Origin %q: status %d, want %d", origin, resp.StatusCode, want)
		}
	}

	t.Run("same origin is accepted", func(t *testing.T) {
		send(t, ts.URL, http.StatusNoContent)
	})
	t.Run("no origin header is accepted", func(t *testing.T) {
		// Same-origin requests and non-browser clients often omit it, and
		// rejecting those breaks more than it protects.
		send(t, "", http.StatusNoContent)
	})
	t.Run("another origin is refused", func(t *testing.T) {
		send(t, "https://evil.example", http.StatusForbidden)
	})
	t.Run("a lookalike host is refused", func(t *testing.T) {
		send(t, ts.URL+".evil.example", http.StatusForbidden)
	})
	t.Run("a malformed origin is refused", func(t *testing.T) {
		send(t, "://:::", http.StatusForbidden)
	})

	t.Run("AllowOrigin overrides it", func(t *testing.T) {
		custom, cts := newTestServer(t, Options{
			AllowOrigin: func(origin string, _ *http.Request) bool {
				return origin == "https://trusted.example"
			},
		})
		csess := newSession(custom)
		custom.On("act", func(c *Ctx) error { return nil })

		req, _ := http.NewRequest(http.MethodPost, cts.URL+"/_alacris/live",
			strings.NewReader(`{"p":"`+csess.ID()+`","a":"act","d":null}`))
		req.AddCookie(sessionCookie(csess))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://trusted.example")
		resp, err := cts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("status %d, want 204", resp.StatusCode)
		}
	})
}
