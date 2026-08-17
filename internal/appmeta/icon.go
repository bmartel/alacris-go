package appmeta

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

func pngToICO(pngBytes []byte) ([]byte, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("appmeta: icon is not a PNG: %w", err)
	}
	w, h := cfg.Width, cfg.Height
	if w > 256 {
		w = 0
	}
	if h > 256 {
		h = 0
	}
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	buf.WriteByte(byte(w))
	buf.WriteByte(byte(h))
	buf.WriteByte(0)
	buf.WriteByte(0)
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(32))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(pngBytes)))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(22))
	buf.Write(pngBytes)
	return buf.Bytes(), nil
}

func pngToICNS(pngBytes []byte) ([]byte, error) {
	if _, err := png.DecodeConfig(bytes.NewReader(pngBytes)); err != nil {
		return nil, fmt.Errorf("appmeta: icon is not a PNG: %w", err)
	}
	chunk := "ic08" // 256×256 PNG
	if len(pngBytes) > 1<<20 {
		chunk = "ic09" // 512×512
	}
	total := 8 + 8 + len(pngBytes)
	var buf bytes.Buffer
	buf.WriteString("icns")
	_ = binary.Write(&buf, binary.BigEndian, uint32(total))
	buf.WriteString(chunk)
	_ = binary.Write(&buf, binary.BigEndian, uint32(8+len(pngBytes)))
	buf.Write(pngBytes)
	return buf.Bytes(), nil
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
