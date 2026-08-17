//go:build !desktop

package app

func requireDesktop() error { return ErrNoDesktop }

func newWindow(Options) (*Window, error) {
	return nil, ErrNoDesktop
}

func applyMenu(*Window, *Menu) {}

func applyTray(*Window, *Tray) {}

func setDeepLinkHandler(func(string)) {}

func registerShortcut(w *Window, keys string, fn func()) error { return ErrNoDesktop }

func unregisterShortcut(w *Window, keys string) error { return ErrNoDesktop }
