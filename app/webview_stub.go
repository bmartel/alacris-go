//go:build !desktop

package app

func requireDesktop() error { return ErrNoDesktop }

func newWindow(Options) (*Window, error) {
	return nil, ErrNoDesktop
}
