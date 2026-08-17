//go:build !darwin && !linux && !windows

package app

import (
	"context"
	"fmt"
)

func openURL(context.Context, string) error {
	return fmt.Errorf("app: OpenURL is not supported on this OS")
}

func reveal(context.Context, string) error {
	return fmt.Errorf("app: RevealInFileManager is not supported on this OS")
}
