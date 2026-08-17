package app

import (
	"context"
	"fmt"
	"os/exec"
)

// OpenURL opens raw in the user's default handler. http, https, mailto, and
// registered app schemes are accepted; other schemes are refused.
func OpenURL(ctx context.Context, raw string) error {
	if !looksLikeURL(raw) {
		return fmt.Errorf("app: OpenURL: not a URL")
	}
	return openURL(ctx, raw)
}

// RevealInFileManager shows path in the OS file manager.
func RevealInFileManager(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("app: RevealInFileManager: empty path")
	}
	return reveal(ctx, path)
}

var openExec = func(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}
