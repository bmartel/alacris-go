//go:build darwin

package app

import "context"

func openURL(ctx context.Context, raw string) error {
	return openExec(ctx, "open", "--", raw)
}

func reveal(ctx context.Context, path string) error {
	return openExec(ctx, "open", "-R", "--", path)
}
