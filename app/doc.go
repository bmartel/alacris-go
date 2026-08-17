// Package app hosts an alacris-go live application in a native OS webview.
//
// The live protocol does not change. Go still renders elements, patches props
// over SSE, and receives component events over POST. This package is a host
// for that model: it serves the same http.Handler on a loopback address and
// opens the operating system's webview onto it. There is no JavaScript
// binding layer, no generated TypeScript, and no second way to talk to Go.
// Native capabilities (dialogs, menus, the filesystem, notifications) are
// ordinary Go function calls, typically from a live.On handler or a menu
// callback.
//
//	app.Run(app.Options{
//	    Title:   "Board",
//	    Width:   1100,
//	    Height:  800,
//	    Handler: mux,
//	    Menu:    app.DefaultMenu(),
//	})
//
// # Build tags
//
// Opening a window requires a binary built with `-tags desktop`. That pulls
// CGO and the OS webview (WKWebView, WebView2, WebKitGTK). Without the tag,
// Listen still works — it is ordinary HTTP — and Run returns ErrNoDesktop.
//
// # Host token
//
// EventSource requires an HTTP family URL, so the webview cannot use a custom
// scheme. Open binds 127.0.0.1 and issues a random host token: the first
// navigation carries it as a query parameter, the gate swaps it for an
// HttpOnly cookie and redirects. Another local process that can reach the
// port cannot create a session without that token. Listen, used by tests,
// does not install the gate. The live cookie is still required.
package app
