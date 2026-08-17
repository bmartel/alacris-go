//go:build linux

package app

import (
	"context"
	"os/exec"
	"strings"
)

func writeClipboard(ctx context.Context, text string) error {
	if _, err := exec.LookPath("wl-copy"); err == nil {
		_, err := clipExec(ctx, "wl-copy", text)
		return err
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		_, err := clipExec(ctx, "xclip", text, "-selection", "clipboard")
		return err
	}
	return clipUnavailable("wl-copy/xclip")
}

func readClipboard(ctx context.Context) (string, error) {
	if _, err := exec.LookPath("wl-paste"); err == nil {
		out, err := clipExec(ctx, "wl-paste", "")
		return strings.TrimRight(out, "\n"), err
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		out, err := clipExec(ctx, "xclip", "", "-selection", "clipboard", "-o")
		return strings.TrimRight(out, "\n"), err
	}
	return "", clipUnavailable("wl-paste/xclip")
}
