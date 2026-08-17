package app

import (
	"context"
	"testing"
)

func TestOpenURLRejectsBarePath(t *testing.T) {
	t.Parallel()
	if err := OpenURL(context.Background(), "/etc/passwd"); err == nil {
		t.Fatal("bare path succeeded")
	}
}

func TestOpenURLInvokesHelper(t *testing.T) {
	orig := openExec
	t.Cleanup(func() { openExec = orig })
	var args []string
	openExec = func(_ context.Context, name string, a ...string) error {
		args = append([]string{name}, a...)
		return nil
	}
	if err := OpenURL(context.Background(), "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if len(args) < 2 {
		t.Fatalf("args = %v", args)
	}
}
