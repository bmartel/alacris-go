//go:build linux

package app

import "context"

func notify(ctx context.Context, n Notification) error {
	title := n.Title
	if title == "" {
		title = "alacris"
	}
	return notifyExec(ctx, "notify-send", title, n.Body)
}
