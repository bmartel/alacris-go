package appmeta

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPngToICOAndICNS(t *testing.T) {
	t.Parallel()
	pngBytes := tinyPNG(t)
	ico, err := pngToICO(pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(ico) < len(pngBytes) {
		t.Fatalf("ico too small: %d", len(ico))
	}
	icns, err := pngToICNS(pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(icns, []byte("icns")) {
		t.Fatal("icns magic")
	}
}

func TestBundleDeepLinkPlist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin := filepath.Join(dir, "in")
	if err := os.WriteFile(bin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	c := Config{Name: "Board", Identifier: "com.example.board", DeepLinkScheme: "board"}
	got, err := bundle(c, bin, out, "darwin")
	if err != nil {
		t.Fatal(err)
	}
	plist, err := os.ReadFile(filepath.Join(got, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plist), "CFBundleURLSchemes") || !strings.Contains(string(plist), "board") {
		t.Fatalf("plist:\n%s", plist)
	}
}

func TestBundleLinuxMime(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bin := filepath.Join(dir, "in")
	if err := os.WriteFile(bin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	c := Config{Name: "Board", Identifier: "com.example.board", DeepLinkScheme: "board"}
	if _, err := bundle(c, bin, out, "linux"); err != nil {
		t.Fatal(err)
	}
	desk, err := os.ReadFile(filepath.Join(out, "share", "applications", "board.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(desk), "x-scheme-handler/board") {
		t.Fatalf("desktop:\n%s", desk)
	}
}

func TestPackSkipsMissingTools(t *testing.T) {
	orig := lookPath
	t.Cleanup(func() { lookPath = orig })
	lookPath = func(string) (string, error) { return "", os.ErrNotExist }
	dir := t.TempDir()
	c := Config{Name: "Board", Identifier: "com.example.board"}
	got, err := Pack(c, dir, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("got %q", got)
	}
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
