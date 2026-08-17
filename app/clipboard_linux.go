//go:build linux

package app

import (
	"context"
	"strings"
)

func writeClipboard(ctx context.Context, text string) error {
	if _, err := clipExec(ctx, "wl-copy", text); err == nil {
		return nil
	}
	if _, err := clipExec(ctx, "xclip", text, "-selection", "clipboard"); err == nil {
		return nil
	}
	return clipUnavailable("wl-copy/xclip")
}

func readClipboard(ctx context.Context) (string, error) {
	if out, err := clipExec(ctx, "wl-paste", ""); err == nil {
		return strings.TrimRight(out, "\n"), nil
	}
	if out, err := clipExec(ctx, "xclip", "", "-selection", "clipboard", "-o"); err == nil {
		return strings.TrimRight(out, "\n"), nil
	}
	return "", clipUnavailable("wl-paste/xclip")
}
