package appmeta

import (
	"bytes"
	"encoding/binary"
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

// The chunk type is the size macOS uses the image at, so it has to describe
// the image actually stored. Choosing it by file size declared a 1024x1024
// icon as ic08 — the 256 slot — and shipped one entry where the Dock wants a
// ladder.
func TestICNSDeclaresTheSizeItStores(t *testing.T) {
	t.Parallel()
	icns, err := pngToICNS(squarePNG(t, 1024))
	if err != nil {
		t.Fatal(err)
	}
	got := icnsChunks(t, icns)
	if len(got) < 6 {
		t.Fatalf("%d entries, want the ladder: %v", len(got), got)
	}
	for typ, size := range got {
		want := 0
		for _, e := range icnsEntries {
			if e.typ == typ {
				want = e.size
			}
		}
		if want == 0 {
			t.Errorf("unknown chunk %q", typ)
			continue
		}
		if size != want {
			t.Errorf("chunk %s holds %dx%d, but that type means %d", typ, size, size, want)
		}
	}
	if _, ok := got["ic10"]; !ok {
		t.Errorf("no 1024 entry from a 1024 source: %v", got)
	}
	if _, ok := got["ic11"]; !ok {
		t.Errorf("no 32 entry, so the Dock downsamples the largest: %v", got)
	}
}

// A source smaller than the top of the ladder must not be upscaled into a
// blur: the rungs above it are left out.
func TestICNSSkipsSizesLargerThanTheSource(t *testing.T) {
	t.Parallel()
	icns, err := pngToICNS(squarePNG(t, 128))
	if err != nil {
		t.Fatal(err)
	}
	got := icnsChunks(t, icns)
	for typ, size := range got {
		if size > 128 {
			t.Errorf("chunk %s is %d, larger than the 128 source", typ, size)
		}
	}
	if _, ok := got["ic07"]; !ok {
		t.Errorf("no 128 entry from a 128 source: %v", got)
	}
}

// ICO records a side in one byte, 0 meaning 256, so a larger image cannot be
// described — it has to be scaled rather than announced as something it is not.
func TestICOFitsTheFormat(t *testing.T) {
	t.Parallel()
	ico, err := pngToICO(squarePNG(t, 1024))
	if err != nil {
		t.Fatal(err)
	}
	w, h := ico[6], ico[7]
	if w != 0 || h != 0 {
		t.Errorf("directory says %dx%d, want 0x0 for 256", w, h)
	}
	off := binary.LittleEndian.Uint32(ico[18:22])
	cfg, err := png.DecodeConfig(bytes.NewReader(ico[off:]))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 256 || cfg.Height != 256 {
		t.Errorf("payload is %dx%d, want 256x256", cfg.Width, cfg.Height)
	}
}

// Averaging colour and alpha separately drags the colour of fully transparent
// pixels into the edge. A transparent black surround must not darken an
// opaque white square when it is scaled down.
func TestScalingDoesNotFringeTransparentEdges(t *testing.T) {
	t.Parallel()
	src := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			i := src.PixOffset(x, y)
			// Deliberately not on a multiple of the 4:1 scale, so that the
			// destination pixel at x=4 straddles the boundary and has to
			// blend rather than land wholly inside the square.
			if x >= 18 && x < 46 && y >= 18 && y < 46 {
				src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = 0xff, 0xff, 0xff, 0xff
			}
			// else: transparent black, the default.
		}
	}
	dst := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	scaleBox(dst, src)
	i := dst.PixOffset(8, 8) // dead centre, all-white source
	if dst.Pix[i] != 0xff || dst.Pix[i+1] != 0xff || dst.Pix[i+2] != 0xff {
		t.Errorf("centre is %v, want opaque white", dst.Pix[i:i+4])
	}
	// The boundary pixel is half white and half transparent: the colour must
	// stay white and only the alpha fall.
	e := dst.PixOffset(4, 8)
	if dst.Pix[e+3] == 0xff || dst.Pix[e+3] == 0 {
		t.Fatalf("edge alpha %d, expected partial", dst.Pix[e+3])
	}
	if dst.Pix[e] < 0xf0 {
		t.Errorf("edge colour %v is fringed dark, want white", dst.Pix[e:e+4])
	}
}

func icnsChunks(t *testing.T, b []byte) map[string]int {
	t.Helper()
	if string(b[:4]) != "icns" {
		t.Fatal("icns magic")
	}
	if n := binary.BigEndian.Uint32(b[4:8]); int(n) != len(b) {
		t.Fatalf("header says %d bytes, file is %d", n, len(b))
	}
	out := map[string]int{}
	for off := 8; off < len(b); {
		typ := string(b[off : off+4])
		n := int(binary.BigEndian.Uint32(b[off+4 : off+8]))
		if n < 8 || off+n > len(b) {
			t.Fatalf("chunk %q length %d at %d", typ, n, off)
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(b[off+8 : off+n]))
		if err != nil {
			t.Fatalf("chunk %q is not a PNG: %v", typ, err)
		}
		out[typ] = cfg.Width
		off += n
	}
	return out
}

func squarePNG(t *testing.T, side int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 0x40, 0x20, 0x80, 0xff
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
