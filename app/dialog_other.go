//go:build !darwin && !windows && !linux

package app

import (
	"context"
	"fmt"
)

func openFile(context.Context, FileDialog) (string, error) {
	return "", fmt.Errorf("app: file dialogs are not supported on this OS")
}

func saveFile(context.Context, FileDialog) (string, error) {
	return "", fmt.Errorf("app: file dialogs are not supported on this OS")
}

func message(context.Context, string, string) error {
	return fmt.Errorf("app: dialogs are not supported on this OS")
}

func confirm(context.Context, string, string) (bool, error) {
	return false, fmt.Errorf("app: dialogs are not supported on this OS")
}
