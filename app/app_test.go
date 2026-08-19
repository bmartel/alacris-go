package app

import (
	"errors"
	"net/http"
	"testing"
)

func TestRunWithoutDesktopTag(t *testing.T) {
	t.Parallel()
	if requireDesktop() == nil {
		t.Skip("built with -tags desktop")
	}
	err := Run(Options{Handler: stubHandler{}})
	if !errors.Is(err, ErrNoDesktop) {
		t.Fatalf("Run without -tags desktop = %v, want ErrNoDesktop", err)
	}
}

type stubHandler struct{}

func (stubHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

// Undecorated predates Titlebar and has to keep working, without either field
// quietly overruling a caller who set the other on purpose.
func TestTitlebarForReconcilesUndecorated(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		opts Options
		want Titlebar
	}{
		{"nothing asked for", Options{}, TitlebarNative},
		{"the old spelling", Options{Undecorated: true}, TitlebarHidden},
		{"the new one", Options{Titlebar: TitlebarInset}, TitlebarInset},
		{"both, and the specific one wins",
			Options{Undecorated: true, Titlebar: TitlebarInset}, TitlebarInset},
		{"asking for the default explicitly",
			Options{Titlebar: TitlebarNative}, TitlebarNative},
	} {
		if got := titlebarFor(c.opts); got != c.want {
			t.Errorf("%s: titlebarFor = %v, want %v", c.name, got, c.want)
		}
	}
}
