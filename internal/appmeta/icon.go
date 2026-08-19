package appmeta

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func writeIcons(c Config, dest string, goos string) error {
	if c.Icon == "" {
		return nil
	}
	b, err := os.ReadFile(c.Icon)
	if err != nil {
		return fmt.Errorf("appmeta: icon: %w", err)
	}
	switch goos {
	case "darwin":
		icns, err := pngToICNS(b)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, "Contents", "Resources", "icon.icns"), icns, 0o644)
	case "windows":
		ico, err := pngToICO(b)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, c.BinaryName()+".ico"), ico, 0o644)
	case "linux":
		iconDir := filepath.Join(dest, "share", "icons", "hicolor", "256x256", "apps")
		if err := os.MkdirAll(iconDir, 0o755); err != nil {
			return err
		}
		return copyFile(c.Icon, filepath.Join(iconDir, c.BinaryName()+".png"), 0o644)
	}
	return nil
}

// icnsEntry is one image in an icns file: the four-byte chunk type and the
// pixel size that type means.
//
// Two types can name the same size on purpose — ic08 is 256 and ic13 is
// 128@2x, which is also 256 pixels. macOS picks by the slot it needs, so
// leaving one out means a display it cannot serve at native resolution.
var icnsEntries = []struct {
	typ  string
	size int
}{
	{"ic11", 32},   // 16@2x
	{"ic12", 64},   // 32@2x
	{"ic07", 128},  // 128
	{"ic13", 256},  // 128@2x
	{"ic08", 256},  // 256
	{"ic14", 512},  // 256@2x
	{"ic09", 512},  // 512
	{"ic10", 1024}, // 512@2x
}

// pngToICNS builds a multi-resolution icns from one PNG.
//
// The chunk type is the size macOS will use the image at, so it has to be
// read off the decoded image rather than guessed. Picking it by file size —
// which is what this did — declared a 1024x1024 icon as ic08, the 256 slot:
// readable, because every reader trusts the payload over the label, but a
// lie, and a single entry at any size leaves the Dock downsampling one large
// bitmap to 16 pixels. The whole ladder is written instead, each rung scaled
// from the source, and rungs larger than the source are skipped rather than
// upscaled into a blur.
func pngToICNS(pngBytes []byte) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("appmeta: icon is not a PNG: %w", err)
	}
	b := src.Bounds()
	side := b.Dx()
	if b.Dy() < side {
		side = b.Dy()
	}

	var body bytes.Buffer
	written := 0
	for _, e := range icnsEntries {
		if e.size > side && written > 0 {
			break
		}
		var payload []byte
		if e.size == side {
			payload = pngBytes
		} else {
			payload, err = encodeScaled(src, e.size)
			if err != nil {
				return nil, err
			}
		}
		body.WriteString(e.typ)
		_ = binary.Write(&body, binary.BigEndian, uint32(8+len(payload)))
		body.Write(payload)
		written++
	}

	var buf bytes.Buffer
	buf.WriteString("icns")
	_ = binary.Write(&buf, binary.BigEndian, uint32(8+body.Len()))
	buf.Write(body.Bytes())
	return buf.Bytes(), nil
}

// pngToICO wraps a PNG as a single-image icon.
//
// The ICO directory records a size in one byte, with 0 standing for 256, so
// anything larger cannot be described at all — a 1024 icon handed over whole
// is announced as 256 and drawn from a payload four times that. It is scaled
// down to fit the format instead.
func pngToICO(pngBytes []byte) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("appmeta: icon is not a PNG: %w", err)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > 256 || h > 256 {
		pngBytes, err = encodeScaled(src, 256)
		if err != nil {
			return nil, err
		}
		w, h = 256, 256
	}

	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	buf.WriteByte(byte(w % 256)) // 256 is recorded as 0
	buf.WriteByte(byte(h % 256))
	buf.WriteByte(0)
	buf.WriteByte(0)
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(pngBytes)))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(22))
	buf.Write(pngBytes)
	return buf.Bytes(), nil
}

// encodeScaled resamples to a square of the given side and encodes a PNG.
//
// A box filter — every destination pixel is the average of the source pixels
// it covers — rather than nearest neighbour, which at these ratios turns a
// hairline into a dotted line or drops it entirely. Alpha is premultiplied
// before averaging, because averaging colour and alpha separately pulls the
// colour of fully transparent pixels into the edges as a dark fringe.
func encodeScaled(src image.Image, side int) ([]byte, error) {
	rgba := image.NewNRGBA(image.Rect(0, 0, side, side))
	scaleBox(rgba, src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, fmt.Errorf("appmeta: scaling the icon: %w", err)
	}
	return buf.Bytes(), nil
}

func scaleBox(dst *image.NRGBA, src image.Image) {
	sb := src.Bounds()
	dw, dh := dst.Bounds().Dx(), dst.Bounds().Dy()
	if dw == 0 || dh == 0 || sb.Empty() {
		return
	}
	// Upscaling has no source pixels to average; hand it to the stdlib.
	if sb.Dx() <= dw || sb.Dy() <= dh {
		draw.Draw(dst, dst.Bounds(), src, sb.Min, draw.Src)
		return
	}
	for y := 0; y < dh; y++ {
		y0 := sb.Min.Y + y*sb.Dy()/dh
		y1 := sb.Min.Y + (y+1)*sb.Dy()/dh
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < dw; x++ {
			x0 := sb.Min.X + x*sb.Dx()/dw
			x1 := sb.Min.X + (x+1)*sb.Dx()/dw
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, a uint64
			n := uint64((x1 - x0) * (y1 - y0))
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					// RGBA() is already alpha-premultiplied.
					pr, pg, pb, pa := src.At(sx, sy).RGBA()
					r += uint64(pr)
					g += uint64(pg)
					b += uint64(pb)
					a += uint64(pa)
				}
			}
			r, g, b, a = r/n, g/n, b/n, a/n
			// Back out of premultiplied for NRGBA.
			var nr, ng, nb uint64
			if a > 0 {
				nr, ng, nb = r*0xffff/a, g*0xffff/a, b*0xffff/a
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i+0] = uint8(min64(nr, 0xffff) >> 8)
			dst.Pix[i+1] = uint8(min64(ng, 0xffff) >> 8)
			dst.Pix[i+2] = uint8(min64(nb, 0xffff) >> 8)
			dst.Pix[i+3] = uint8(a >> 8)
		}
	}
}

func min64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func iconPlistName(c Config) string {
	if c.Icon == "" {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(c.Icon), filepath.Ext(c.Icon))
	if strings.EqualFold(filepath.Ext(c.Icon), ".png") {
		return "icon"
	}
	return base
}
