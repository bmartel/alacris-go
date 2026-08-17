//go:build desktop

package app

import (
	"fmt"

	webview "github.com/webview/webview_go"
)

func requireDesktop() error { return nil }

func newWindow(opts Options) (*Window, error) {
	w := webview.New(opts.Dev)
	if w == nil {
		return nil, fmt.Errorf("app: failed to create webview")
	}
	return &Window{impl: &nativeWindow{w: w}}, nil
}

type nativeWindow struct {
	w webview.WebView
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

func (n *nativeWindow) setFixedSize(on bool) {
	if on {
		// HintFixed needs a size; the last SetSize stands.
		width, height := 800, 600
		n.w.SetSize(width, height, webview.HintFixed)
	}
}

func (n *nativeWindow) minimize() {
	// webview_go has no minimize; closing is the supported control.
}

func (n *nativeWindow) close()            { n.w.Terminate() }
func (n *nativeWindow) navigate(u string) { n.w.Navigate(u) }
func (n *nativeWindow) initJS(js string)  { n.w.Init(js) }
func (n *nativeWindow) run() error {
	n.w.Run()
	return nil
}
func (n *nativeWindow) destroy() { n.w.Destroy() }
