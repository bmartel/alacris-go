package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrNoDesktop is returned by Run when the binary was not built with
// `-tags desktop`. Listen does not need the tag.
var ErrNoDesktop = errors.New("app: this binary was built without desktop support; rebuild with -tags desktop")

// ErrCanceled is returned by a dialog when the user dismissed it.
var ErrCanceled = errors.New("app: canceled")

// Appearance is the window chrome colour scheme. The page still follows
// whatever Config.Theme (or the OS) says; this only hints the title bar.
type Appearance int

const (
	// AppearanceSystem follows the OS. It is the default.
	AppearanceSystem Appearance = iota
	AppearanceLight
	AppearanceDark
)

// Options configure a desktop window around an http.Handler.
type Options struct {
	// Title is the window title. Defaults to "alacris".
	Title string

	// Width and Height are the initial content size in pixels.
	// They default to 800×600.
	Width  int
	Height int

	// MinWidth and MinHeight bound shrinking. Zero means no minimum.
	MinWidth  int
	MinHeight int

	// FixedSize prevents the user from resizing the window.
	FixedSize bool

	// Handler is the same mux a browser build would ListenAndServe.
	// Required.
	Handler http.Handler

	// Path is the URL path the webview opens, relative to the loopback
	// origin. Defaults to "/".
	Path string

	// Addr is the loopback address to bind. Empty means 127.0.0.1:0
	// (a free port). Only loopback hosts are accepted; 0.0.0.0 is refused.
	Addr string

	// Menu is installed as the native menu bar where the backend supports
	// it (currently macOS). Nil means no menu; DefaultMenu is the usual
	// Edit/Quit set.
	Menu *Menu

	// Appearance hints the title bar. The default follows the OS.
	Appearance Appearance

	// Dev opens the webview inspector.
	Dev bool

	// OnReady runs after the window exists and the loopback server is
	// listening, immediately before the native event loop blocks. Window
	// methods that do not need Dispatch (SetTitle, SetSize, Close) are
	// safe here.
	OnReady func(*Window)
}

func (o *Options) fill() error {
	if o.Handler == nil {
		return fmt.Errorf("app: Options.Handler is required")
	}
	if o.Title == "" {
		o.Title = "alacris"
	}
	if o.Width <= 0 {
		o.Width = 800
	}
	if o.Height <= 0 {
		o.Height = 600
	}
	if o.Path == "" {
		o.Path = "/"
	} else if !strings.HasPrefix(o.Path, "/") {
		o.Path = "/" + o.Path
	}
	return nil
}

// An App owns the loopback host and, once Open has run, a window.
type App struct {
	opts Options
	host *Host
	win  *Window
}

// New prepares an App. It does not listen or open a window.
func New(opts Options) *App {
	return &App{opts: opts}
}

// Run listens on loopback, opens a native window onto the handler, and
// blocks until the window closes.
func Run(opts Options) error {
	return New(opts).Run()
}

// Run opens a window if needed and blocks in the native event loop.
func (a *App) Run() error {
	if err := requireDesktop(); err != nil {
		return err
	}
	if a.win == nil {
		if _, err := a.Open(); err != nil {
			return err
		}
	}
	defer a.closeHost()
	return a.win.run()
}

// Open starts the loopback server and creates the window without entering
// the event loop. A second window is not supported yet; a second Open
// returns the existing one.
func (a *App) Open() (*Window, error) {
	if a.win != nil {
		return a.win, nil
	}
	if err := a.opts.fill(); err != nil {
		return nil, err
	}
	h, err := Listen(a.opts)
	if err != nil {
		return nil, err
	}
	a.host = h
	w, err := newWindow(a.opts)
	if err != nil {
		_ = h.Shutdown(context.Background())
		a.host = nil
		return nil, err
	}
	w.host = h
	w.SetTitle(a.opts.Title)
	w.SetSize(a.opts.Width, a.opts.Height)
	if a.opts.MinWidth > 0 || a.opts.MinHeight > 0 {
		w.setMinSize(a.opts.MinWidth, a.opts.MinHeight)
	}
	if a.opts.FixedSize {
		w.setFixedSize(true)
	}
	applyMenu(w, a.opts.Menu)
	w.navigate(h.URL() + a.opts.Path)
	applyAppearance(w, a.opts.Appearance)
	a.win = w
	if a.opts.OnReady != nil {
		a.opts.OnReady(w)
	}
	return w, nil
}

// Window is the window Open created, or nil.
func (a *App) Window() *Window { return a.win }

// Host is the loopback server Open created, or nil.
func (a *App) Host() *Host { return a.host }

func (a *App) closeHost() {
	if a.host == nil {
		return
	}
	_ = a.host.Shutdown(context.Background())
	a.host = nil
}
