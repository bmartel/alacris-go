//go:build windows

package app

import (
	"context"
	"fmt"
	"strings"
)

func notify(ctx context.Context, n Notification) error {
	title := powershellQuote(n.Title)
	body := powershellQuote(n.Body)
	ps := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
$xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$xml.GetElementsByTagName('text').Item(0).AppendChild($xml.CreateTextNode(%s)) | Out-Null
$xml.GetElementsByTagName('text').Item(1).AppendChild($xml.CreateTextNode(%s)) | Out-Null
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('alacris').Show($toast)
`, title, body)
	return notifyExec(ctx, "powershell", "-NoProfile", "-Command", ps)
}

func powershellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
