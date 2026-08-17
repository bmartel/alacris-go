//go:build desktop

package app

import (
	"fmt"
	"unsafe"

	webview "github.com/webview/webview_go"
)

func requireDesktop() error { return nil }

func newWindow(opts Options) (*Window, error) {
	w := webview.New(opts.Dev)
	if w == nil {
		return nil, fmt.Errorf("app: failed to create webview")
	}
	return &Window{impl: &nativeWindow{w: w, hwnd: w.Window()}}, nil
}

type nativeWindow struct {
	w    webview.WebView
	hwnd unsafe.Pointer
}

func (n *nativeWindow) nativeHandle() uintptr {
	if n == nil || n.hwnd == nil {
		return 0
	}
	return uintptr(n.hwnd)
}

func (n *nativeWindow) setTitle(title string) { n.w.SetTitle(title) }

func (n *nativeWindow) setSize(w, h int) {
	n.w.SetSize(w, h, webview.HintNone)
}

func (n *nativeWindow) setMinSize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	n.w.SetSize(w, h, webview.HintMin)
}

func (n *nativeWindow) setMaxSize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	n.w.SetSize(w, h, webview.HintMax)
}

func (n *nativeWindow) setFixedSize(on bool) {
	if on {
		width, height := 800, 600
		n.w.SetSize(width, height, webview.HintFixed)
	}
}

func (n *nativeWindow) setPosition(x, y int) { windowSetPosition(n.hwnd, x, y) }
func (n *nativeWindow) position() (int, int) { return windowPosition(n.hwnd) }
func (n *nativeWindow) center()              { windowCenter(n.hwnd) }
func (n *nativeWindow) minimize()            { windowMinimize(n.hwnd) }
func (n *nativeWindow) maximize()            { windowMaximize(n.hwnd) }
func (n *nativeWindow) unmaximize()          { windowUnmaximize(n.hwnd) }
func (n *nativeWindow) fullscreen()          { windowFullscreen(n.hwnd) }
func (n *nativeWindow) unfullscreen()        { windowUnfullscreen(n.hwnd) }
func (n *nativeWindow) show()                { windowShow(n.hwnd) }
func (n *nativeWindow) hide()                { windowHide(n.hwnd) }
func (n *nativeWindow) focus()               { windowFocus(n.hwnd) }
func (n *nativeWindow) setAlwaysOnTop(on bool) {
	windowSetAlwaysOnTop(n.hwnd, on)
}
func (n *nativeWindow) setDecorations(on bool) {
	windowSetDecorations(n.hwnd, on)
}
func (n *nativeWindow) setBadge(s string) { windowSetBadge(s) }

func (n *nativeWindow) close()            { n.w.Terminate() }
func (n *nativeWindow) navigate(u string) { n.w.Navigate(u) }
func (n *nativeWindow) initJS(js string)  { n.w.Init(js) }
func (n *nativeWindow) run() error {
	n.w.Run()
	return nil
}
func (n *nativeWindow) destroy() { n.w.Destroy() }
