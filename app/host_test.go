package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLoopbackBindRejectsNonLoopback(t *testing.T) {
	t.Parallel()
	for _, addr := range []string{"0.0.0.0:0", ":8080", "[::]:80", "192.168.1.1:9", "example.com:80"} {
		if _, err := loopbackBind(addr); err == nil {
			t.Errorf("loopbackBind(%q) succeeded, want error", addr)
		}
	}
}

func TestLoopbackBindAcceptsLoopback(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":               "127.0.0.1:0",
		"127.0.0.1:0":    "127.0.0.1:0",
		"127.0.0.1:8080": "127.0.0.1:8080",
		"localhost:9":    "127.0.0.1:9",
		"[::1]:0":        "[::1]:0",
	}
	for in, want := range cases {
		got, err := loopbackBind(in)
		if err != nil {
			t.Errorf("loopbackBind(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("loopbackBind(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestListenBindsLoopbackAndServes(t *testing.T) {
	t.Parallel()
	h, err := Listen(Options{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	host, port, err := net.SplitHostPort(strings.TrimPrefix(h.URL(), "http://"))
	if err != nil {
		t.Fatal(err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		t.Fatalf("bound %q, want a loopback IP", host)
	}
	if _, err := strconv.Atoi(port); err != nil {
		t.Fatalf("port %q: %v", port, err)
	}

	res, err := http.Get(h.URL() + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("GET / = %d, want 204", res.StatusCode)
	}
}

func TestListenRejectsWrongHostHeader(t *testing.T) {
	t.Parallel()
	h, err := Listen(Options{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	req, err := http.NewRequest(http.MethodGet, h.URL()+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "evil.example"
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("spoofed Host = %d, want 403", res.StatusCode)
	}
}

func TestListenRejectsNonLoopbackPeer(t *testing.T) {
	t.Parallel()
	if !remoteLoopback("127.0.0.1:1234") {
		t.Fatal("127.0.0.1 should be loopback")
	}
	if !remoteLoopback("[::1]:9") {
		t.Fatal("::1 should be loopback")
	}
	if remoteLoopback("8.8.8.8:443") {
		t.Fatal("8.8.8.8 must not be loopback")
	}
	if remoteLoopback("192.168.0.10:80") {
		t.Fatal("RFC1918 must not count as loopback")
	}
}

func TestHostAllowed(t *testing.T) {
	t.Parallel()
	want := "127.0.0.1:54321"
	if !hostAllowed("127.0.0.1:54321", want) {
		t.Error("exact Host should match")
	}
	if hostAllowed("127.0.0.1:9", want) {
		t.Error("wrong port should not match")
	}
	if hostAllowed("localhost:54321", want) {
		t.Error("localhost should not match a 127.0.0.1 bind")
	}
	if hostAllowed("evil.example", want) {
		t.Error("foreign Host should not match")
	}
}

func TestListenRequiresHandler(t *testing.T) {
	t.Parallel()
	if _, err := Listen(Options{}); err == nil {
		t.Fatal("Listen with no Handler succeeded")
	}
}

func TestListenBootstrapURLHasNoToken(t *testing.T) {
	t.Parallel()
	h, err := Listen(Options{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })
	u := h.BootstrapURL("/")
	if strings.Contains(u, hostQuery) {
		t.Fatalf("ungated BootstrapURL leaked a token: %s", u)
	}
}

func TestShutdownStopsServing(t *testing.T) {
	t.Parallel()
	h, err := Listen(Options{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = http.Get(h.URL() + "/")
	if err == nil {
		t.Fatal("GET after Shutdown succeeded")
	}
}

func TestShutdownDoesNotWaitOnAStream(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	h, err := Listen(Options{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "no flush", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher.Flush()
			close(started)
			<-r.Context().Done()
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		res, err := http.Get(h.URL() + "/")
		if err != nil {
			return
		}
		defer res.Body.Close()
		io.Copy(io.Discard, res.Body)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("stream never started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_ = h.Shutdown(ctx)
	if took := time.Since(start); took > time.Second {
		t.Fatalf("Shutdown took %s waiting on a stream that never ends", took)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stream handler did not return after Shutdown")
	}
}
