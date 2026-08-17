//go:build windows

package app

import "context"

func openURL(ctx context.Context, raw string) error {
	return openExec(ctx, "rundll32", "url.dll,FileProtocolHandler", raw)
}

func reveal(ctx context.Context, path string) error {
	return openExec(ctx, "explorer", "/select,"+path)
}
