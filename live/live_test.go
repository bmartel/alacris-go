package live

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestServer(t *testing.T, opts ...Options) (*Server, *httptest.Server) {
	t.Helper()
	o := Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if len(opts) > 0 {
		o = opts[0]
		if o.Logger == nil {
			o.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
		}
	}
	srv := New(o)
	ts := httptest.NewServer(http.StripPrefix("/_alacris", srv))
	t.Cleanup(func() {
		ts.Close()
		srv.Close()
	})
	return srv, ts
}

// stream opens an SSE connection and returns a channel of decoded frames.
func stream(t *testing.T, ts *httptest.Server, sess *Session) (<-chan []Patch, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		ts.URL+"/_alacris/live?p="+sess.ID(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(sessionCookie(sess))
	resp, err := ts.Client().Do(req)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		t.Fatalf("stream status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("content type %q", ct)
	}

	frames := make(chan []Patch, 16)
	go func() {
		defer close(frames)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			line := sc.Text()
			data, ok := strings.CutPrefix(line, "data: ")
			if !ok {
				continue
			}
			var patches []Patch
			if err := json.Unmarshal([]byte(data), &patches); err != nil {
				return
			}
			frames <- patches
		}
	}()

	return frames, func() {
		cancel()
		resp.Body.Close()
	}
}

func recv(t *testing.T, frames <-chan []Patch) []Patch {
	t.Helper()
	select {
	case f, ok := <-frames:
		if !ok {
			t.Fatal("stream closed while waiting for a frame")
		}
		return f
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for a frame")
		return nil
	}
}

func TestPropPatchReachesTheBrowser(t *testing.T) {
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	frames, stop := stream(t, ts, sess)
	defer stop()

	// Give the stream a moment to subscribe, then send.
	waitConnected(t, sess)
	sess.Element("cart").Set("count", 3)

	got := recv(t, frames)
	if len(got) != 1 {
		t.Fatalf("got %d patches, want 1", len(got))
	}
	want := Patch{Op: OpProp, ID: "cart", Key: "count", Value: float64(3)}
	if got[0] != want {
		t.Errorf("got %+v, want %+v", got[0], want)
	}
}

func TestPatchesBeforeConnectAreBuffered(t *testing.T) {
	// A page renders, then the server changes something, then the browser
	// finally connects. Nothing may be lost in that window.
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	sess.Element("cart").Set("count", 1)
	sess.Element("cart").Set("count", 2)

	frames, stop := stream(t, ts, sess)
	defer stop()

	got := recv(t, frames)
	if len(got) != 2 {
		t.Fatalf("got %d patches, want the 2 that were buffered", len(got))
	}
	if got[1].Value != float64(2) {
		t.Errorf("last value = %v, want 2", got[1].Value)
	}
}

func TestBufferDropsOldestRatherThanBlocking(t *testing.T) {
	srv, _ := newTestServer(t, Options{Buffer: 4})
	sess := newSession(srv)

	for i := 0; i < 100; i++ {
		sess.Element("x").Set("n", i)
	}

	sess.mu.Lock()
	pending := append([]Patch(nil), sess.pending...)
	dropped := sess.dropped
	sess.mu.Unlock()

	if len(pending) != 4 {
		t.Fatalf("buffered %d patches, want 4", len(pending))
	}
	// The newest survive: for a stream of writes to one prop, the last value
	// is the one the page needs.
	if pending[3].Value != 99 {
		t.Errorf("newest buffered value = %v, want 99", pending[3].Value)
	}
	if !dropped {
		t.Error("dropping patches should be recorded")
	}
}

func TestBatchIsOneFrame(t *testing.T) {
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	frames, stop := stream(t, ts, sess)
	defer stop()
	waitConnected(t, sess)

	sess.Batch(func() {
		sess.Element("a").Set("x", 1)
		sess.Element("b").Set("y", 2)
		sess.Element("c").Class("on", true)
	})

	got := recv(t, frames)
	if len(got) != 3 {
		t.Fatalf("got %d patches in the first frame, want all 3", len(got))
	}
}

func TestOnOpenRunsOnEveryConnect(t *testing.T) {
	// A reconnecting EventSource missed whatever happened while it was away,
	// so the server has to be able to restate the world.
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	var mu sync.Mutex
	opens := 0
	sess.OnOpen(func(s *Session) {
		mu.Lock()
		opens++
		mu.Unlock()
		s.Element("cart").Set("count", 7)
	})

	for i := 0; i < 2; i++ {
		frames, stop := stream(t, ts, sess)
		got := recv(t, frames)
		if len(got) != 1 || got[0].Value != float64(7) {
			t.Fatalf("connect %d: got %+v", i, got)
		}
		stop()
		waitDisconnected(t, sess)
	}

	mu.Lock()
	defer mu.Unlock()
	if opens != 2 {
		t.Errorf("OnOpen ran %d times, want 2", opens)
	}
}

func TestSecondConnectionReplacesTheFirst(t *testing.T) {
	// A reload can open the new stream before the old one has finished dying.
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	first, stopFirst := stream(t, ts, sess)
	defer stopFirst()
	waitConnected(t, sess)

	second, stopSecond := stream(t, ts, sess)
	defer stopSecond()

	select {
	case _, open := <-first:
		if open {
			t.Error("the replaced stream should be closed, not fed")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the first stream was never closed")
	}

	waitConnected(t, sess)
	sess.Element("x").Set("n", 1)
	if got := recv(t, second); len(got) != 1 {
		t.Errorf("the new stream got %d patches, want 1", len(got))
	}
}

func TestActionRoundTrip(t *testing.T) {
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	type addDetail struct {
		Text string `json:"text"`
	}

	On(srv, "add-todo", func(c *Ctx, d addDetail) error {
		if c.Element != "todos" {
			t.Errorf("element = %q, want todos", c.Element)
		}
		c.Handle().Set("items", []string{d.Text})
		return nil
	})

	frames, stop := stream(t, ts, sess)
	defer stop()
	waitConnected(t, sess)

	post(t, ts, sess, `{"p":"`+sess.ID()+`","a":"add-todo","i":"todos","d":{"text":"buy milk"}}`, http.StatusNoContent)

	got := recv(t, frames)
	if len(got) != 1 || got[0].Key != "items" {
		t.Fatalf("got %+v", got)
	}
	values, ok := got[0].Value.([]any)
	if !ok || len(values) != 1 || values[0] != "buy milk" {
		t.Errorf("value = %#v", got[0].Value)
	}
}

func TestSessionHandlerBeatsServerHandler(t *testing.T) {
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	srv.On("act", func(c *Ctx) error {
		t.Error("the server-wide handler should not have run")
		return nil
	})
	ran := make(chan struct{}, 1)
	sess.On("act", func(c *Ctx) error {
		ran <- struct{}{}
		return nil
	})

	post(t, ts, sess, `{"p":"`+sess.ID()+`","a":"act","i":"","d":null}`, http.StatusNoContent)

	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("the session handler never ran")
	}
}

func TestActionRejections(t *testing.T) {
	srv, ts := newTestServer(t, Options{MaxDetail: 64})
	sess := newSession(srv)
	srv.On("act", func(c *Ctx) error { return nil })

	t.Run("unknown session", func(t *testing.T) {
		post(t, ts, sess, `{"p":"not-a-session","a":"act","d":null}`, http.StatusNotFound)
	})

	t.Run("unknown action", func(t *testing.T) {
		post(t, ts, sess, `{"p":"`+sess.ID()+`","a":"nope","d":null}`, http.StatusNotFound)
	})

	t.Run("oversized payload", func(t *testing.T) {
		big := `{"p":"` + sess.ID() + `","a":"act","d":{"t":"` + strings.Repeat("x", 500) + `"}}`
		post(t, ts, sess, big, http.StatusRequestEntityTooLarge)
	})

	t.Run("cross-origin", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/_alacris/live",
			strings.NewReader(`{"p":"`+sess.ID()+`","a":"act","d":null}`))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(sessionCookie(sess))
		req.Header.Set("Origin", "https://evil.example")
		req.Header.Set("Content-Type", "application/json")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("status %d, want 403", resp.StatusCode)
		}
	})

}

func TestActionDetailMustFitItsType(t *testing.T) {
	// A component emits what it says it emits; a console can emit anything.
	// Binding is strict so a payload that does not match the declared detail
	// is refused rather than quietly arriving half-filled.
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	type detail struct {
		Text string `json:"text"`
	}
	On(srv, "typed", func(c *Ctx, d detail) error {
		t.Errorf("the handler ran with %+v", d)
		return nil
	})

	post(t, ts, sess, `{"p":"`+sess.ID()+`","a":"typed","d":{"unexpected":1}}`, http.StatusInternalServerError)
}

func TestUnknownSessionStream(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := ts.Client().Get(ts.URL + "/_alacris/live?s=nope")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status %d, want 404", resp.StatusCode)
	}
}

func TestSessionExpiry(t *testing.T) {
	now := time.Now()
	var mu sync.Mutex
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}

	srv := New(Options{
		TTL:    time.Second,
		Now:    clock,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	defer srv.Close()

	sess := newSession(srv)
	if sess.expired(now, time.Second) {
		t.Error("a fresh session should not be expired")
	}

	mu.Lock()
	now = now.Add(2 * time.Second)
	future := now
	mu.Unlock()

	if !sess.expired(future, time.Second) {
		t.Error("a session with no browser should expire after its TTL")
	}

	sess.Close()
	if _, ok := srv.Session(sess.ID()); ok {
		t.Error("a closed session should be gone from the server")
	}
	// Sending to a closed session is a no-op, not a panic: the page it was
	// for is gone, and the caller has nothing useful to do about it.
	sess.Element("x").Set("n", 1)
}

func TestSessionContextOutlivesTheRequest(t *testing.T) {
	// The mistake this prevents: capturing the rendering request's context and
	// using it from OnOpen. That callback runs when the browser attaches, by
	// which time the request has finished and its context is cancelled — so
	// every SetHTML from it failed, silently, on the server.
	srv, ts := newTestServer(t)
	sess := newSession(srv)

	if err := sess.Context().Err(); err != nil {
		t.Fatalf("a fresh session's context is already done: %v", err)
	}

	// Stand in for the request that rendered the page: finished, cancelled.
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	cancelRequest()

	rendered := make(chan error, 2)
	sess.OnOpen(func(s *Session) {
		rendered <- s.Element("chip").SetHTML(s.Context(), "", templText("<b>4 left</b>"))
		rendered <- s.Element("chip").SetHTML(requestCtx, "", templText("<b>4 left</b>"))
	})

	frames, stop := stream(t, ts, sess)
	defer stop()

	if err := <-rendered; err != nil {
		t.Errorf("the session's own context should still be live: %v", err)
	}
	if err := <-rendered; err == nil {
		t.Error("the finished request's context should not have worked; the test is not proving anything")
	}

	if got := recv(t, frames); len(got) != 1 || got[0].Op != OpHTML {
		t.Errorf("got %+v, want one OpHTML patch", got)
	}

	sess.Close()
	if err := sess.Context().Err(); err == nil {
		t.Error("closing a session should cancel its context")
	}
}

func TestSessionIDsAreUnguessable(t *testing.T) {
	srv := New(Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	defer srv.Close()

	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := newSession(srv).ID()
		if seen[id] {
			t.Fatalf("duplicate session id %q", id)
		}
		if len(id) < 20 {
			t.Fatalf("session id %q is too short to be a capability", id)
		}
		seen[id] = true
	}
}

func TestClientScript(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := ts.Client().Get(ts.URL + "/_alacris/live.js")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
		t.Errorf("content type %q", ct)
	}
	for _, want := range []string{"EventSource", "composedPath", "data-ala-on"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("the client script is missing %q", want)
		}
	}
}

func TestPatchEncoding(t *testing.T) {
	// The client reads these fields by name, so the encoding is an interface
	// and not an implementation detail.
	cases := map[string]struct {
		patch Patch
		want  string
	}{
		"prop": {
			Prop("cart", "count", 3),
			`{"o":"p","i":"cart","k":"count","v":3}`,
		},
		// A false or zero prop value must survive: it is the value, not the
		// absence of one.
		"false prop": {
			Prop("cart", "open", false),
			`{"o":"p","i":"cart","k":"open","v":false}`,
		},
		"zero prop": {
			Prop("cart", "count", 0),
			`{"o":"p","i":"cart","k":"count","v":0}`,
		},
		"empty string prop": {
			Prop("cart", "label", ""),
			`{"o":"p","i":"cart","k":"label","v":""}`,
		},
		// Removing an attribute is a nil value.
		"attribute removal": {
			Attr("cart", "hidden", nil),
			`{"o":"a","i":"cart","k":"hidden"}`,
		},
		// The default slot's name is "", and it has to be on the wire, or the
		// client cannot tell it apart from a missing key. Angle brackets come
		// out escaped, which is what keeps markup from ending the SSE frame.
		"default slot": {
			Patch{Op: OpHTML, ID: "chip", Key: "", Value: "<b>hi</b>"},
			`{"o":"h","i":"chip","k":"","v":"<b>hi</b>"}`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			body, err := json.Marshal(c.patch)
			if err != nil {
				t.Fatal(err)
			}
			// Compare the decoded objects: Go escapes angle brackets in
			// strings and JSON.parse undoes that, so the bytes differ from the
			// expectation while the frame the client sees does not. Which keys
			// are present is still compared exactly.
			var got, want map[string]any
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(c.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("\n got: %v\nwant: %v\n(raw: %s)", got, want, body)
			}
		})
	}
}

func TestHTMLPatchRenders(t *testing.T) {
	p, err := HTML(context.Background(), "card", "title", templText("<b>hi</b>"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Op != OpHTML || p.ID != "card" || p.Key != "title" || p.Value != "<b>hi</b>" {
		t.Errorf("got %+v", p)
	}
}

// post sends an action as the browser holding sess would. A nil session means
// no cookie, which is how a request that has not been authorised looks.
func post(t *testing.T, ts *httptest.Server, sess *Session, body string, wantStatus int) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/_alacris/live", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if sess != nil {
		req.AddCookie(sessionCookie(sess))
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != wantStatus {
		t.Errorf("status %d, want %d", resp.StatusCode, wantStatus)
	}
}

func waitConnected(t *testing.T, s *Session) {
	t.Helper()
	waitFor(t, s.Connected, "the browser never attached")
}

func waitDisconnected(t *testing.T, s *Session) {
	t.Helper()
	waitFor(t, func() bool { return !s.Connected() }, "the browser never detached")
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal(msg)
}

func TestBindEmptyStructHandlesPrimitiveDetail(t *testing.T) {
	c := &Ctx{
		Action: "click",
		Detail: json.RawMessage("1"), // MouseEvent numeric detail
	}
	var empty struct{}
	if err := c.Bind(&empty); err != nil {
		t.Fatalf("Bind to empty struct failed with numeric detail: %v", err)
	}

	cString := &Ctx{
		Action: "custom",
		Detail: json.RawMessage(`"hello"`),
	}
	if err := cString.Bind(&empty); err != nil {
		t.Fatalf("Bind to empty struct failed with string detail: %v", err)
	}

	// Strictly checks non-empty struct
	type TypedDetail struct {
		Count int `json:"count"`
	}
	var typed TypedDetail
	if err := c.Bind(&typed); err == nil {
		t.Fatalf("Bind to typed struct should have failed on invalid number literal")
	}
}
