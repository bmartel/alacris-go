package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
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
		var args []string
		_ = json.NewDecoder(io.LimitReader(c, 4096)).Decode(&args)
		done <- args
	}()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if err := pingInstance(port, []string{"a", "b"}); err != nil {
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
