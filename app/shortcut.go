package app

import "errors"

// ErrNoShortcut means this OS cannot register a global hotkey.
var ErrNoShortcut = errors.New("app: global shortcuts are not available")

// RegisterShortcut binds keys (the same grammar as MenuItem.Keys) as a
// global hotkey while the app is running. Needs `-tags desktop`.
func (w *Window) RegisterShortcut(keys string, fn func()) error {
	if w == nil {
		return ErrNoDesktop
	}
	return registerShortcut(w, keys, fn)
}

// UnregisterShortcut drops a previously registered hotkey.
func (w *Window) UnregisterShortcut(keys string) error {
	if w == nil {
		return ErrNoDesktop
	}
	return unregisterShortcut(w, keys)
}
