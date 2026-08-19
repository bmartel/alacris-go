package app

// A Window is one native webview. An App may open more than one; Run
// blocks on the first until it closes.
type Window struct {
	impl windowImpl
	host *Host
	app  *App
}

type windowImpl interface {
	setTitle(string)
	setSize(w, h int)
	setMinSize(w, h int)
	setMaxSize(w, h int)
	setFixedSize(bool)
	setPosition(x, y int)
	position() (x, y int)
	center()
	minimize()
	maximize()
	unmaximize()
	fullscreen()
	unfullscreen()
	show()
	hide()
	focus()
	setAlwaysOnTop(bool)
	setDecorations(bool)
	setTitlebar(Titlebar)
	beginDrag()
	setBadge(string)
	close()
	navigate(url string)
	initJS(js string)
	run() error
	destroy()
	nativeHandle() uintptr
}

// URL is the loopback origin this window was navigated to.
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

func (w *Window) setMaxSize(width, height int) {
	if w != nil && w.impl != nil {
		w.impl.setMaxSize(width, height)
	}
}

func (w *Window) setFixedSize(on bool) {
	if w != nil && w.impl != nil {
		w.impl.setFixedSize(on)
	}
}

// SetPosition moves the window's top-left corner, in screen pixels.
func (w *Window) SetPosition(x, y int) {
	if w != nil && w.impl != nil {
		w.impl.setPosition(x, y)
	}
}

// Position is the window's top-left corner, in screen pixels.
func (w *Window) Position() (x, y int) {
	if w != nil && w.impl != nil {
		return w.impl.position()
	}
	return 0, 0
}

// Center places the window on the primary display.
func (w *Window) Center() {
	if w != nil && w.impl != nil {
		w.impl.center()
	}
}

// Minimize hides the window in the dock / taskbar.
func (w *Window) Minimize() {
	if w != nil && w.impl != nil {
		w.impl.minimize()
	}
}

// Maximize fills the work area. Unmaximize restores the previous size.
func (w *Window) Maximize() {
	if w != nil && w.impl != nil {
		w.impl.maximize()
	}
}

// Unmaximize restores a maximized window.
func (w *Window) Unmaximize() {
	if w != nil && w.impl != nil {
		w.impl.unmaximize()
	}
}

// Fullscreen enters the OS full-screen space.
func (w *Window) Fullscreen() {
	if w != nil && w.impl != nil {
		w.impl.fullscreen()
	}
}

// Unfullscreen leaves full-screen.
func (w *Window) Unfullscreen() {
	if w != nil && w.impl != nil {
		w.impl.unfullscreen()
	}
}

// Show makes a hidden window visible.
func (w *Window) Show() {
	if w != nil && w.impl != nil {
		w.impl.show()
	}
}

// Hide removes the window from the screen without destroying it.
func (w *Window) Hide() {
	if w != nil && w.impl != nil {
		w.impl.hide()
	}
}

// Focus brings the window to the front.
func (w *Window) Focus() {
	if w != nil && w.impl != nil {
		w.impl.focus()
	}
}

// SetAlwaysOnTop keeps the window above others when on is true.
func (w *Window) SetAlwaysOnTop(on bool) {
	if w != nil && w.impl != nil {
		w.impl.setAlwaysOnTop(on)
	}
}

// SetDecorations shows or hides the native title bar.
func (w *Window) SetDecorations(on bool) {
	if w != nil && w.impl != nil {
		w.impl.setDecorations(on)
	}
}

// SetBadge sets the dock / taskbar badge. Empty clears it. No-op where
// the OS has no badge.
// SetTitlebar chooses how the title bar is drawn.
func (w *Window) SetTitlebar(style Titlebar) {
	if w == nil || w.impl == nil {
		return
	}
	w.impl.setTitlebar(style)
}

// BeginDrag starts moving the window, as though the pointer had grabbed a
// title bar.
//
// It is for a page that draws its own title bar: call it from a pointerdown in
// the region that should behave like one, over the live layer. The OS then
// runs the drag, so the window keeps up with the pointer rather than chasing
// it through a round trip per frame.
//
// It only does anything while a mouse button is actually down, because every
// platform implements it by handing the current event back to the window
// manager.
func (w *Window) BeginDrag() {
	if w == nil || w.impl == nil {
		return
	}
	w.impl.beginDrag()
}

func (w *Window) SetBadge(s string) {
	if w != nil && w.impl != nil {
		w.impl.setBadge(s)
	}
}

// Close terminates this window. Closing the last window ends Run.
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

func (w *Window) handle() uintptr {
	if w == nil || w.impl == nil {
		return 0
	}
	return w.impl.nativeHandle()
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
