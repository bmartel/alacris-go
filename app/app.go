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

// Titlebar is how the window's title bar is drawn.
//
// The default is the OS one. The other two exist because an application whose
// own surface runs to the top edge looks wrong underneath a bar the OS painted
// a different colour, and there is no way to tint that bar: on macOS it is
// AppKit's, in whatever grey the current appearance says.
type Titlebar int

const (
	// TitlebarNative is the OS title bar, drawn by the OS. The default.
	TitlebarNative Titlebar = iota

	// TitlebarInset keeps the window buttons and lets the page draw behind
	// them. Content runs to the top edge and the bar is whatever the page
	// paints there; dragging and the buttons stay native.
	//
	// This is what a modern desktop application usually wants. On macOS it is
	// a transparent title bar over a full-size content view, so the traffic
	// lights float above the page. Windows and Linux have no equivalent of
	// that arrangement, so they fall back to TitlebarHidden and the page is
	// expected to draw its own buttons.
	TitlebarInset

	// TitlebarHidden removes the title bar altogether. The page draws its own,
	// and BeginDrag moves the window.
	TitlebarHidden
)

// titlebarFor reconciles the two ways of asking for the same thing.
//
// Undecorated came first and means TitlebarHidden. An explicit Titlebar wins,
// so setting both is not a contradiction to resolve at runtime — the newer,
// more specific field simply says more.
func titlebarFor(opts Options) Titlebar {
	if opts.Titlebar == TitlebarNative && opts.Undecorated {
		return TitlebarHidden
	}
	return opts.Titlebar
}

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

	// Menu is installed as the native menu bar on macOS, Windows, and
	// Linux. Nil means no menu; DefaultMenu is the usual Edit/Quit set.
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

	// X and Y are the initial window origin in screen pixels. Ignored
	// when Center is true. Both zero means the OS picks.
	X, Y int

	// Center places the window on the primary display. Overrides X and Y.
	Center bool

	// MaxWidth and MaxHeight bound growing. Zero means no maximum.
	MaxWidth  int
	MaxHeight int

	// Fullscreen opens the window in the OS full-screen space.
	Fullscreen bool

	// Undecorated hides the native title bar. Equivalent to
	// Titlebar: TitlebarHidden, which is the clearer spelling.
	Undecorated bool

	// Titlebar chooses how the title bar is drawn. The zero value is the OS
	// one, so this is opt-in.
	Titlebar Titlebar

	// AlwaysOnTop keeps the window above others.
	AlwaysOnTop bool

	// Hidden opens the window without showing it (a tray-only start).
	Hidden bool

	// Identifier is the reverse-DNS id used for single-instance locking,
	// data directories, and deep-link registration. Required when
	// SingleInstance is true.
	Identifier string

	// SingleInstance refuses a second process and forwards its arguments
	// to OnSecondInstance on the first.
	SingleInstance bool

	// OnSecondInstance runs in the first process when another launch is
	// refused. Args are the second process's os.Args[1:].
	OnSecondInstance func(args []string)

	// DeepLinkScheme registers an URL scheme (no "://") the OS should
	// deliver to this app. OnDeepLink receives the full URL.
	DeepLinkScheme string

	// OnDeepLink runs when the OS delivers a matching URL. Also invoked
	// for a second-instance launch whose args look like a URL.
	OnDeepLink func(url string)

	// Tray, if set, installs a status-item / notification-area icon.
	Tray *Tray

	// OnClose is asked before the last window goes away. Returning false
	// cancels the close (the window stays). Nil means allow.
	OnClose func() bool
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

// An App owns the loopback host and, once Open has run, one or more windows.
type App struct {
	opts     Options
	host     *Host
	win      *Window
	wins     []*Window
	inst     *instanceLock
	deeplink func(string)
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
	if err := FinishPendingUpdate(); err != nil {
		return err
	}
	if a.win == nil {
		if _, err := a.Open(); err != nil {
			return err
		}
	}
	defer a.closeHost()
	defer a.releaseInstance()
	return a.win.run()
}

// Open starts the gated loopback server and creates the first window
// without entering the event loop. A second Open returns the existing
// first window; use NewWindow for another.
func (a *App) Open() (*Window, error) {
	if a.win != nil {
		return a.win, nil
	}
	if err := a.opts.fill(); err != nil {
		return nil, err
	}
	if err := a.acquireInstance(); err != nil {
		return nil, err
	}
	if a.opts.DeepLinkScheme != "" {
		a.deeplink = a.opts.OnDeepLink
		registerDeepLink(a.opts.DeepLinkScheme, a.opts.Identifier, func(u string) {
			if a.deeplink != nil {
				a.deeplink(u)
			}
		})
	}
	h, err := listen(a.opts, true)
	if err != nil {
		a.releaseInstance()
		return nil, err
	}
	a.host = h
	w, err := a.createWindow()
	if err != nil {
		_ = h.Shutdown(context.Background())
		a.host = nil
		a.releaseInstance()
		return nil, err
	}
	a.win = w
	if a.opts.Tray != nil {
		applyTray(w, a.opts.Tray)
	}
	if a.opts.OnReady != nil {
		a.opts.OnReady(w)
	}
	return w, nil
}

// NewWindow opens another window onto the same host. Open must have run.
func (a *App) NewWindow() (*Window, error) {
	if a.host == nil {
		return a.Open()
	}
	return a.createWindow()
}

func (a *App) createWindow() (*Window, error) {
	w, err := newWindow(a.opts)
	if err != nil {
		return nil, err
	}
	w.host = a.host
	w.app = a
	w.SetTitle(a.opts.Title)
	w.SetSize(a.opts.Width, a.opts.Height)
	if a.opts.MinWidth > 0 || a.opts.MinHeight > 0 {
		w.setMinSize(a.opts.MinWidth, a.opts.MinHeight)
	}
	if a.opts.MaxWidth > 0 || a.opts.MaxHeight > 0 {
		w.setMaxSize(a.opts.MaxWidth, a.opts.MaxHeight)
	}
	if a.opts.FixedSize {
		w.setFixedSize(true)
	}
	if style := titlebarFor(a.opts); style != TitlebarNative {
		w.SetTitlebar(style)
	}
	if a.opts.AlwaysOnTop {
		w.SetAlwaysOnTop(true)
	}
	if a.opts.Center {
		w.Center()
	} else if a.opts.X != 0 || a.opts.Y != 0 {
		w.SetPosition(a.opts.X, a.opts.Y)
	}
	applyMenu(w, a.opts.Menu)
	w.navigate(a.host.BootstrapURL(a.opts.Path))
	applyAppearance(w, a.opts.Appearance)
	if a.opts.Fullscreen {
		w.Fullscreen()
	}
	if a.opts.Hidden {
		w.Hide()
	}
	a.wins = append(a.wins, w)
	return w, nil
}

func (a *App) acquireInstance() error {
	if !a.opts.SingleInstance {
		return nil
	}
	if a.opts.Identifier == "" {
		return fmt.Errorf("app: SingleInstance requires Options.Identifier")
	}
	onSecond := a.opts.OnSecondInstance
	lock, err := acquireInstance(a.opts.Identifier, osArgs(), func(args []string) {
		if len(args) == 1 && looksLikeURL(args[0]) && a.opts.OnDeepLink != nil {
			a.opts.OnDeepLink(args[0])
			return
		}
		if onSecond != nil {
			onSecond(args)
		}
		if a.win != nil {
			a.win.Focus()
		}
	})
	if err != nil {
		return err
	}
	a.inst = lock
	return nil
}

func (a *App) releaseInstance() {
	if a.inst != nil {
		a.inst.release()
		a.inst = nil
	}
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
