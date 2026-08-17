package app

import (
	"context"
	"runtime"
	"testing"
)

func TestOpenURLRejectsBarePath(t *testing.T) {
	t.Parallel()
	if err := OpenURL(context.Background(), "/etc/passwd"); err == nil {
		t.Fatal("bare path succeeded")
	}
}

func TestOpenURLInvokesHelper(t *testing.T) {
	orig := openExec
	t.Cleanup(func() { openExec = orig })
	var args []string
	openExec = func(_ context.Context, name string, a ...string) error {
		args = append([]string{name}, a...)
		return nil
	}
	if err := OpenURL(context.Background(), "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if len(args) < 2 {
		t.Fatalf("args = %v", args)
	}
}

// A web page that can influence the OpenURL argument must not reach a local
// file, a network share, or a script scheme, and must not smuggle a leading
// dash past the opener as an option.
func TestOpenURLRefusesDangerousSchemes(t *testing.T) {
	t.Parallel()
	orig := openExec
	t.Cleanup(func() { openExec = orig })
	openExec = func(context.Context, string, ...string) error {
		t.Fatal("opener ran for a URL that should have been refused")
		return nil
	}
	for _, raw := range []string{
		"file:///etc/passwd",
		"FILE://x",
		"smb://server/share",
		"javascript://%0aalert(1)",
		"-x://y",
		"://nohost",
	} {
		if err := OpenURL(context.Background(), raw); err == nil {
			t.Errorf("OpenURL(%q) was allowed", raw)
		}
	}
}

// The validated URL still reaches the opener behind a "--", so even a value
// that survives validation cannot be read as an option. Only the openers that
// accept it (open, xdg-open) get one; rundll32 on Windows does not, and there
// the scheme check alone rejects a leading dash.
func TestOpenURLPassesEndOfOptions(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("opener on %s does not take --", runtime.GOOS)
	}
	orig := openExec
	t.Cleanup(func() { openExec = orig })
	var args []string
	openExec = func(_ context.Context, _ string, a ...string) error {
		args = a
		return nil
	}
	if err := OpenURL(context.Background(), "https://example.com"); err != nil {
		t.Fatal(err)
	}
	sawSep := false
	for i, a := range args {
		if a == "--" && i < len(args)-1 {
			sawSep = true
		}
	}
	if !sawSep {
		t.Errorf("opener args %v lack a '--' before the URL", args)
	}
}
