//go:build windows

package app

import (
	"context"
	"strings"
)

func writeClipboard(ctx context.Context, text string) error {
	_, err := clipExec(ctx, "powershell", text, "-NoProfile", "-Command", "Set-Clipboard -Value ([Console]::In.ReadToEnd())")
	return err
}

func readClipboard(ctx context.Context) (string, error) {
	out, err := clipExec(ctx, "powershell", "", "-NoProfile", "-Command", "Get-Clipboard")
	return strings.TrimRight(out, "\r\n"), err
}
