package app

// A Tray is a status-item / notification-area icon. Menu items with Do
// run as Go callbacks, the same as the window menu.
type Tray struct {
	// Title is shown when the OS cannot draw an icon (macOS extra, Linux).
	Title string
	// Tooltip is the hover text.
	Tooltip string
	Menu    *Menu
}
