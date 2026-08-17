package app

// A Window is one native webview. v1 ships a single window per App.
type Window struct {
	impl windowImpl
	host *Host
}

type windowImpl interface {
	setTitle(string)
	setSize(w, h int)
	setMinSize(w, h int)
	setFixedSize(bool)
	minimize()
	close()
	navigate(url string)
	initJS(js string)
	run() error
	destroy()
}

// URL is the loopback origin plus path this window was navigated to.
func (w *Window) URL() string {
	if w == nil || w.host == nil {
		return ""
	}
	return w.host.URL()
}

// SetTitle updates the native title bar.
func (w *Window) SetTitle(title string) {
	if w != nil && w.impl != nil {
		w.impl.setTitle(title)
	}
}

// SetSize updates the content size in pixels.
func (w *Window) SetSize(width, height int) {
	if w != nil && w.impl != nil {
		w.impl.setSize(width, height)
	}
}

func (w *Window) setMinSize(width, height int) {
	if w != nil && w.impl != nil {
		w.impl.setMinSize(width, height)
	}
}

func (w *Window) setFixedSize(on bool) {
	if w != nil && w.impl != nil {
		w.impl.setFixedSize(on)
	}
}

// Minimize hides the window in the dock / taskbar.
func (w *Window) Minimize() {
	if w != nil && w.impl != nil {
		w.impl.minimize()
	}
}

// Close terminates the native event loop. Run then returns.
func (w *Window) Close() {
	if w != nil && w.impl != nil {
		w.impl.close()
	}
}

func (w *Window) navigate(url string) {
	if w != nil && w.impl != nil {
		w.impl.navigate(url)
	}
}

func (w *Window) run() error {
	if w == nil || w.impl == nil {
		return ErrNoDesktop
	}
	defer w.impl.destroy()
	return w.impl.run()
}

func applyAppearance(w *Window, a Appearance) {
	if w == nil || w.impl == nil {
		return
	}
	switch a {
	case AppearanceLight:
		w.impl.initJS(`document.documentElement.style.colorScheme='light'`)
	case AppearanceDark:
		w.impl.initJS(`document.documentElement.style.colorScheme='dark'`)
	}
}
