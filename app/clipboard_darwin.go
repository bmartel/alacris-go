//go:build darwin

package app

import (
	"context"
	"strings"
)

func writeClipboard(ctx context.Context, text string) error {
	_, err := clipExec(ctx, "pbcopy", text)
	return err
}

func readClipboard(ctx context.Context) (string, error) {
	out, err := clipExec(ctx, "pbpaste", "")
	return strings.TrimRight(out, "\n"), err
}
