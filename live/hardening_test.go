package live

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// A patch whose key would let a state write execute script is dropped before
// it reaches the wire, and the safe patches around it still arrive.
func TestUnsafePatchKeysAreDropped(t *testing.T) {
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	frames, stop := stream(t, ts, sess)
	defer stop()
	waitConnected(t, sess)

	sess.Send(
		Prop("x", "innerHTML", "<img src=x onerror=alert(1)>"),
		Attr("x", "onclick", "alert(1)"),
		Attr("x", "srcdoc", "<script>alert(1)</script>"),
		Attr("x", "not a name", "v"),
		Prop("x", "count", 1),
	)

	got := recv(t, frames)
	if len(got) != 1 {
		t.Fatalf("got %d patches, want only the safe one", len(got))
	}
	if got[0].Key != "count" {
		t.Errorf("surviving patch key = %q, want count", got[0].Key)
	}
}

func TestUnsafePatch(t *testing.T) {
	unsafe := []Patch{
		Prop("x", "innerHTML", "v"),
		Prop("x", "outerHTML", "v"),
		Prop("x", "__proto__", "v"),
		Prop("x", "constructor", "v"),
		Prop("x", "prototype", "v"),
		Attr("x", "onclick", "v"),
		Attr("x", "onPointerDown", "v"),
		Attr("x", "srcdoc", "v"),
		Attr("x", "bad name", "v"),
	}
	for _, p := range unsafe {
		if unsafePatch(p) == "" {
			t.Errorf("patch %+v was allowed", p)
		}
	}
	safe := []Patch{
		Prop("x", "count", 1),
		Prop("x", "onLine", true), // a prop write, not an attribute; define() owns the name
		Attr("x", "aria-busy", "true"),
		Attr("x", "on-line", "v"),
		Class("x", "on", true),
		Reload(),
	}
	for _, p := range safe {
		if reason := unsafePatch(p); reason != "" {
			t.Errorf("patch %+v was refused: %s", p, reason)
		}
	}
}

// An origin is scheme, host and port; an authorisation check must not accept
// two of the three.
func TestOriginSchemeMustMatch(t *testing.T) {
	srv, ts := newTestServer(t)
	sess := newSession(srv)
	srv.On("act", func(c *Ctx) error { return nil })

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	send := func(origin string) int {
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/_alacris/live",
			strings.NewReader(`{"p":"`+sess.ID()+`","a":"act","d":null}`))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(sessionCookie(sess))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	// The test server is plain http, so the https origin has the right host
	// and the wrong scheme.
	if got := send("https://" + u.Host); got != http.StatusForbidden {
		t.Errorf("an https origin drove an http endpoint: status %d, want 403", got)
	}
	if got := send("http://" + u.Host); got != http.StatusNoContent {
		t.Errorf("the genuine origin was refused: status %d, want 204", got)
	}
}

// A zero-value Server has no collector and no session table; the panic should
// say so instead of blaming a nil map.
// A Send racing a reconnect, a detach, or Close must never land on a closed
// channel. Every path that closes the subscriber channel nils the field under
// the session lock, and deliver sends under that same lock — this test is why.
// It once panicked "send on closed channel": Send captured the channel under
// the lock but sent after releasing it, racing the close. Run with -race.
func TestSendRacingReconnectDoesNotPanic(t *testing.T) {
	srv := New(Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	defer srv.Close()
	sess := newSession(srv)

	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					sess.Send(Prop("x", "count", 1))
					sess.Batch(func() { sess.Send(Prop("x", "count", 2)) })
				}
			}
		}()
	}

	// Churn attach/detach so closes constantly race in-flight deliveries.
	// Draining a frame or two first keeps the channel near-full, which is
	// what pushes senders into the racy select arm.
	for i := 0; i < 500; i++ {
		frames, _, release, err := sess.Subscribe()
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-frames:
		default:
		}
		release()
	}
	close(done)
	wg.Wait()
}

func TestZeroValueServerFailsLoudly(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a zero Server did not panic")
		}
		if !strings.Contains(r.(string), "live.New") {
			t.Errorf("panic %q does not name the fix", r)
		}
	}()
	var srv Server
	srv.On("act", func(c *Ctx) error { return nil })
}
