//go:build !darwin && !linux && !windows

package app

import (
	"context"
	"fmt"
)

func notify(context.Context, Notification) error {
	return fmt.Errorf("app: notifications are not supported on this OS")
}
