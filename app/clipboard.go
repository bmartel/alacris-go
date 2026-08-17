package app

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// WriteClipboard puts text on the OS clipboard.
func WriteClipboard(ctx context.Context, text string) error {
	return writeClipboard(ctx, text)
}

// ReadClipboard returns the current OS clipboard text.
func ReadClipboard(ctx context.Context) (string, error) {
	return readClipboard(ctx)
}

var clipExec = func(ctx context.Context, name string, stdin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.Output()
	return string(out), err
}

func clipUnavailable(name string) error {
	return fmt.Errorf("app: clipboard helper %s not found", name)
}
