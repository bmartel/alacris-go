package app

import (
	"context"
	"os/exec"
)

// A Notification is a desktop banner. It does not need `-tags desktop`.
type Notification struct {
	Title string
	Body  string
}

// Notify shows an OS notification.
func Notify(ctx context.Context, n Notification) error {
	return notify(ctx, n)
}

var notifyExec = func(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}
