//go:build !darwin && !linux && !windows

package app

import (
	"context"
	"fmt"
)

func writeClipboard(context.Context, string) error {
	return fmt.Errorf("app: clipboard is not supported on this OS")
}

func readClipboard(context.Context) (string, error) {
	return "", fmt.Errorf("app: clipboard is not supported on this OS")
}
