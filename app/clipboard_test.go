package app

import (
	"context"
	"testing"
)

func TestWriteClipboardInvokesHelper(t *testing.T) {
	orig := clipExec
	t.Cleanup(func() { clipExec = orig })
	var name, stdin string
	clipExec = func(_ context.Context, n string, in string, _ ...string) (string, error) {
		name, stdin = n, in
		return "", nil
	}
	if err := WriteClipboard(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if name == "" || stdin != "hello" {
		t.Fatalf("name=%q stdin=%q", name, stdin)
	}
}
