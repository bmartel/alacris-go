package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSingleInstance(t *testing.T) {
	id := fmt.Sprintf("dev.alacris.test-instance-%d", time.Now().UnixNano())
	got := make(chan []string, 1)
	first, err := acquireInstance(id, nil, func(args []string) { got <- args })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		first.release()
		if dir, err := DataDir(id); err == nil {
			_ = os.RemoveAll(dir)
		}
	})

	_, err = acquireInstance(id, []string{"--other"}, nil)
	if err != ErrAlreadyRunning {
		t.Fatalf("second acquire = %v, want ErrAlreadyRunning", err)
	}
	select {
	case args := <-got:
		if len(args) != 1 || args[0] != "--other" {
			t.Fatalf("args = %v", args)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second-instance handover")
	}
}

func TestLooksLikeURL(t *testing.T) {
	t.Parallel()
	if !looksLikeURL("myapp://open/1") {
		t.Fatal("scheme URL")
	}
	if looksLikeURL("--flag") {
		t.Fatal("flag is not a URL")
	}
}

func TestPingInstanceRoundTrip(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	done := make(chan []string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		r := bufio.NewReader(c)
		gotSecret, _ := r.ReadString('\n')
		if strings.TrimSpace(gotSecret) != "s3cr3t" {
			return
		}
		var args []string
		_ = json.NewDecoder(r).Decode(&args)
		done <- args
	}()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if err := pingInstance(port, "s3cr3t", []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	select {
	case args := <-done:
		if len(args) != 2 || args[0] != "a" {
			t.Fatalf("args = %v", args)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

// A local co-tenant who reaches the loopback port but cannot read the 0600
// lock file — so does not hold the secret — must not be able to drive the
// second-instance handler.
func TestSecondInstanceHandlerRejectsWrongSecret(t *testing.T) {
	id := fmt.Sprintf("dev.alacris.test-instance-secret-%d", time.Now().UnixNano())
	fired := make(chan []string, 1)
	first, err := acquireInstance(id, nil, func(args []string) { fired <- args })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		first.release()
		if dir, err := DataDir(id); err == nil {
			_ = os.RemoveAll(dir)
		}
	})

	// Connect directly with the wrong secret, as a co-tenant with only the
	// port would.
	_, port, err := net.SplitHostPort(first.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if err := pingInstance(port, "not-the-secret", []string{"myapp://evil"}); err != nil {
		t.Fatal(err)
	}
	select {
	case args := <-fired:
		t.Fatalf("handler ran for an unauthenticated peer: %v", args)
	case <-time.After(300 * time.Millisecond):
	}

	// The real secret still works.
	if err := pingInstance(port, first.secret, []string{"myapp://ok"}); err != nil {
		t.Fatal(err)
	}
	select {
	case args := <-fired:
		if len(args) != 1 || args[0] != "myapp://ok" {
			t.Fatalf("args = %v", args)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("authenticated ping did not reach the handler")
	}
}
