//go:build linux

package app

import (
	"context"
	"path/filepath"
)

func openURL(ctx context.Context, raw string) error {
	return openExec(ctx, "xdg-open", "--", raw)
}

func reveal(ctx context.Context, path string) error {
	return openExec(ctx, "xdg-open", "--", filepath.Dir(path))
}
