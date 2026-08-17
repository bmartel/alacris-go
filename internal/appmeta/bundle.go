package appmeta

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// BundleDir writes the OS-specific wrapper around an already-built binary
// and returns the path the user should distribute.
func BundleDir(c Config, binaryPath, destDir string) (string, error) {
	return bundle(c, binaryPath, destDir, runtime.GOOS)
}

func bundle(c Config, binaryPath, destDir, goos string) (string, error) {
	c.fill()
	if err := c.validate(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	switch goos {
	case "darwin":
		return writeMacApp(c, binaryPath, destDir)
	case "linux":
		return writeLinux(c, binaryPath, destDir)
	case "windows":
		return writeWindows(c, binaryPath, destDir)
	default:
		out := filepath.Join(destDir, filepath.Base(binaryPath))
		if err := copyFile(binaryPath, out, 0o755); err != nil {
			return "", err
		}
		return out, nil
	}
}

func writeMacApp(c Config, binaryPath, destDir string) (string, error) {
	appName := c.Name + ".app"
	root := filepath.Join(destDir, appName)
	macOS := filepath.Join(root, "Contents", "MacOS")
	res := filepath.Join(root, "Contents", "Resources")
	if err := os.MkdirAll(macOS, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(res, 0o755); err != nil {
		return "", err
	}
	exe := filepath.Join(macOS, c.BinaryName())
	if err := copyFile(binaryPath, exe, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(root, "Contents", "Info.plist"), []byte(infoPlist(c)), 0o644); err != nil {
		return "", err
	}
	if c.Icon != "" {
		if err := copyFile(c.Icon, filepath.Join(res, filepath.Base(c.Icon)), 0o644); err != nil {
			return "", fmt.Errorf("appmeta: icon: %w", err)
		}
	}
	return root, nil
}

func writeLinux(c Config, binaryPath, destDir string) (string, error) {
	binDir := filepath.Join(destDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}
	exe := filepath.Join(binDir, c.BinaryName())
	if err := copyFile(binaryPath, exe, 0o755); err != nil {
		return "", err
	}
	apps := filepath.Join(destDir, "share", "applications")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(apps, c.BinaryName()+".desktop"), []byte(desktopFile(c)), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(destDir, "nfpm.yaml"), []byte(nfpmFile(c)), 0o644); err != nil {
		return "", err
	}
	return destDir, nil
}

func writeWindows(c Config, binaryPath, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	exe := filepath.Join(destDir, c.BinaryName()+".exe")
	if err := copyFile(binaryPath, exe, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(destDir, "winres.json"), []byte(winresFile(c)), 0o644); err != nil {
		return "", err
	}
	return exe, nil
}

func infoPlist(c Config) string {
	icon := ""
	if c.Icon != "" {
		base := strings.TrimSuffix(filepath.Base(c.Icon), filepath.Ext(c.Icon))
		icon = fmt.Sprintf("  <key>CFBundleIconFile</key>\n  <string>%s</string>\n", base)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleExecutable</key>
  <string>%s</string>
  <key>CFBundleIdentifier</key>
  <string>%s</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>%s</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>%s</string>
  <key>CFBundleVersion</key>
  <string>%s</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
%s</dict>
</plist>
`, xmlEscape(c.BinaryName()), xmlEscape(c.Identifier), xmlEscape(c.Name), xmlEscape(c.Version), xmlEscape(c.Version), icon)
}

func desktopFile(c Config) string {
	cats := strings.Join(c.Bundle.Linux.Categories, ";")
	if cats == "" {
		cats = "Utility;"
	}
	if !strings.HasSuffix(cats, ";") {
		cats += ";"
	}
	exec := c.BinaryName()
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Icon=%s
Categories=%s
StartupWMClass=%s
`, c.Name, exec, c.BinaryName(), cats, c.Identifier)
}

func nfpmFile(c Config) string {
	return fmt.Sprintf(`name: %s
arch: ${GOARCH}
platform: linux
version: %s
maintainer: %s
description: %s
contents:
  - src: bin/%s
    dst: /usr/bin/%s
  - src: share/applications/%s.desktop
    dst: /usr/share/applications/%s.desktop
`, c.BinaryName(), c.Version, c.Identifier, c.Name, c.BinaryName(), c.BinaryName(), c.BinaryName(), c.BinaryName())
}

func winresFile(c Config) string {
	desc := c.Bundle.Windows.FileDescription
	if desc == "" {
		desc = c.Name
	}
	return fmt.Sprintf(`{
  "RT_VERSION": {
    "#1": {
      "0000": {
        "fixed": {"file_version": "%s", "product_version": "%s"},
        "info": {
          "0409": {
            "ProductName": %q,
            "FileDescription": %q,
            "ProductVersion": %q,
            "LegalCopyright": %q
          }
        }
      }
    }
  }
}
`, c.Version, c.Version, c.Name, desc, c.Version, c.Copyright)
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func copyFile(src, dst string, mode os.FileMode) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, mode)
}
