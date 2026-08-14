package livetest_test

import (
	"errors"
	"testing"

	"github.com/bmartel/alacris-go/live"
	"github.com/bmartel/alacris-go/live/livetest"
)

type addDetail struct {
	Text string `json:"text"`
}

func TestHandlerPatchesAreRecordedSynchronously(t *testing.T) {
	srv := live.New()
	defer srv.Close()

	live.On(srv, "add", func(c *live.Ctx, d addDetail) error {
		c.Session.Element("todos").Set("items", []string{d.Text})
		c.Session.Element("counts").Set("value", 1)
		return nil
	})

	sess, rec := livetest.NewSession(t, srv)
	if err := livetest.Invoke(t, sess, "add", "todos", addDetail{Text: "milk"}); err != nil {
		t.Fatal(err)
	}

	got := rec.Patches()
	if len(got) != 2 {
		t.Fatalf("recorded %d patches, want 2: %v", len(got), got)
	}
	p, ok := rec.Last("todos", "items")
	if !ok {
		t.Fatal("no patch for todos.items")
	}
	if items, _ := p.Value.([]string); len(items) != 1 || items[0] != "milk" {
		t.Errorf("items = %#v, want [milk]", p.Value)
	}
}

func TestBatchIsOneRecordedFrame(t *testing.T) {
	srv := live.New()
	defer srv.Close()
	sess, rec := livetest.NewSession(t, srv)

	sess.Batch(func() {
		sess.Element("a").Set("x", 1)
		sess.Element("b").Set("y", 2)
	})
	sess.Element("c").Set("z", 3)

	frames := rec.Frames()
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2 (one batch, one single): %v", len(frames), frames)
	}
	if len(frames[0]) != 2 {
		t.Errorf("the batch frame carries %d patches, want 2", len(frames[0]))
	}
}

func TestOnOpenStateArrives(t *testing.T) {
	srv := live.New()
	defer srv.Close()

	// The real order: render mints the session and registers OnOpen; the
	// browser — here, the recorder — attaches afterwards.
	sess := livetest.RenderSession(t, srv)
	sess.OnOpen(func(s *live.Session) {
		s.Element("cart").Set("count", 7)
	})
	sess.Element("early").Set("n", 1) // buffered: nothing is attached yet

	rec := livetest.Attach(t, sess)
	if _, ok := rec.Last("cart", "count"); !ok {
		t.Error("the OnOpen push was not recorded")
	}
	if _, ok := rec.Last("early", "n"); !ok {
		t.Error("the pre-attach patch was not recorded from the backlog")
	}
}

func TestInvokeUnknownActionIsAnError(t *testing.T) {
	srv := live.New()
	defer srv.Close()
	sess, _ := livetest.NewSession(t, srv)

	err := livetest.Invoke(t, sess, "nope", "", nil)
	if !errors.Is(err, live.ErrNoAction) {
		t.Fatalf("err = %v, want ErrNoAction", err)
	}
}

func TestSessionScopedHandlerWins(t *testing.T) {
	srv := live.New()
	defer srv.Close()

	srv.On("who", func(c *live.Ctx) error {
		c.Session.Element("x").Set("owner", "server")
		return nil
	})
	sess, rec := livetest.NewSession(t, srv)
	sess.On("who", func(c *live.Ctx) error {
		c.Session.Element("x").Set("owner", "session")
		return nil
	})

	if err := livetest.Invoke(t, sess, "who", "", nil); err != nil {
		t.Fatal(err)
	}
	p, _ := rec.Last("x", "owner")
	if p.Value != "session" {
		t.Errorf("owner = %v; the session-scoped handler must take precedence", p.Value)
	}
}
