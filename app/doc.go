// Package app hosts an alacris-go live application in a native OS webview.
//
// The live protocol does not change. Go still renders elements, patches props
// over SSE, and receives component events over POST. This package is a host
// for that model: it serves the same http.Handler on a loopback address and
// opens the operating system's webview onto it. There is no JavaScript
// binding layer, no generated TypeScript, and no second way to talk to Go.
// Native capabilities (dialogs, menus, the filesystem) are ordinary Go
// function calls, typically from a live.On handler or a menu callback.
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
// Opening a window requires a binary built with `-tags desktop`. That pull
// CGO and the OS webview (WKWebView, WebView2, WebKitGTK). Without the tag,
// Listen still works — it is ordinary HTTP — and Run returns ErrNoDesktop.
//
// # Loopback
//
// The host binds 127.0.0.1 (never 0.0.0.0) and rejects requests whose Host
// header is not that address, which closes DNS rebinding. Any other process
// on the machine that can reach the port can still create a session; that is
// the same class of bug Electron and Tauri v1 had with localhost. Do not
// treat a desktop build as a substitute for the live cookie.
package app
