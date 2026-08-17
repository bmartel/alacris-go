//go:build linux

package app

import "context"

func notify(ctx context.Context, n Notification) error {
	title := n.Title
	if title == "" {
		title = "alacris"
	}
	// "--" stops the title or body — which can carry app-supplied text — from
	// being read as notify-send options when it begins with a dash.
	return notifyExec(ctx, "notify-send", "--", title, n.Body)
}
