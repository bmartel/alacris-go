package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func registerDeepLink(scheme, identifier string, fn func(string)) {
	if scheme == "" {
		return
	}
	setDeepLinkHandler(fn)
	registerDeepLinkOS(scheme, identifier)
	go handleStartupDeepLink(fn)
}

func handleStartupDeepLink(fn func(string)) {
	if fn == nil {
		return
	}
	for _, a := range os.Args[1:] {
		if looksLikeURL(a) {
			fn(a)
			return
		}
	}
}

func registerDeepLinkOS(scheme, identifier string) {
	registerDeepLinkPlatform(scheme, identifier)
}

func currentExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	got, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe
	}
	return got
}

func runSilent(name string, args ...string) {
	cmd := exec.Command(name, args...)
	_ = cmd.Run()
}

func validScheme(s string) bool {
	if s == "" || strings.ContainsAny(s, ":/ \t") {
		return false
	}
	return true
}

func schemeError(s string) error {
	return fmt.Errorf("app: invalid deep-link scheme %q", s)
}
