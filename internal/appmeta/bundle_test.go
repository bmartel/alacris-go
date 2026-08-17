package appmeta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleMacApp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin := filepath.Join(dir, "in")
	if err := os.WriteFile(bin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	c := Config{Name: "Board", Identifier: "com.example.board", Version: "0.1.0"}
	got, err := bundle(c, bin, out, "darwin")
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(got, "Contents", "MacOS", "board")
	if _, err := os.Stat(exe); err != nil {
		t.Fatal(err)
	}
	plist, err := os.ReadFile(filepath.Join(got, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plist), "com.example.board") {
		t.Fatalf("plist:\n%s", plist)
	}
}

func TestBundleLinux(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin := filepath.Join(dir, "in")
	if err := os.WriteFile(bin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	c := Config{Name: "Board", Identifier: "com.example.board"}
	if _, err := bundle(c, bin, out, "linux"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "bin", "board")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "share", "applications", "board.desktop")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "nfpm.yaml")); err != nil {
		t.Fatal(err)
	}
}

func TestBundleWindows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin := filepath.Join(dir, "in.exe")
	if err := os.WriteFile(bin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	c := Config{Name: "Board", Identifier: "com.example.board", Version: "1.0.0"}
	got, err := bundle(c, bin, out, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, ".exe") {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(filepath.Join(out, "winres.json")); err != nil {
		t.Fatal(err)
	}
}
