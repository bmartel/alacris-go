package app

import (
	"context"
	"testing"
)

func TestNotifyInvokesHelper(t *testing.T) {
	orig := notifyExec
	t.Cleanup(func() { notifyExec = orig })
	var name string
	notifyExec = func(_ context.Context, n string, _ ...string) error {
		name = n
		return nil
	}
	if err := Notify(context.Background(), Notification{Title: "t", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if name == "" {
		t.Fatal("notify helper was not called")
	}
}
