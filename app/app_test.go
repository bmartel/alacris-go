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
