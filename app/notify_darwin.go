//go:build darwin

package app

import (
	"context"
	"fmt"
)

func notify(ctx context.Context, n Notification) error {
	script := fmt.Sprintf("display notification %s with title %s",
		osaString(n.Body), osaString(n.Title))
	return notifyExec(ctx, "osascript", "-e", script)
}
