// Package appmeta is the desktop app metadata and bundler used by
// `alacris-go app`. It does not import the webview; the CLI stays in the
// root module.
package appmeta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FileName = "alacris.app.json"

// Config is bundle metadata. Window size and live wiring stay in Go.
type Config struct {
	Name           string         `json:"name"`
	Identifier     string         `json:"identifier"`
	Version        string         `json:"version"`
	Main           string         `json:"main,omitempty"`
	Icon           string         `json:"icon,omitempty"`
	Copyright      string         `json:"copyright,omitempty"`
	Category       string         `json:"category,omitempty"`
	DeepLinkScheme string         `json:"deepLinkScheme,omitempty"`
	Bundle         Bundle         `json:"bundle,omitempty"`
	Updater        *UpdaterConfig `json:"updater,omitempty"`
}

// Bundle holds per-OS packaging options.
type Bundle struct {
	MacOS   MacOSBundle   `json:"macOS,omitempty"`
	Windows WindowsBundle `json:"windows,omitempty"`
	Linux   LinuxBundle   `json:"linux,omitempty"`
}

// MacOSBundle is written into Info.plist and used by `app build` on Darwin.
type MacOSBundle struct {
	SigningIdentity string `json:"signingIdentity,omitempty"`
	Notarize        bool   `json:"notarize,omitempty"`
}

// WindowsBundle names how Windows resources are produced.
type WindowsBundle struct {
	FileDescription string `json:"fileDescription,omitempty"`
}

// LinuxBundle names the .desktop categories and package formats.
type LinuxBundle struct {
	Desktop    string   `json:"desktop,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

// UpdaterConfig is copied into the generated latest.json stub.
type UpdaterConfig struct {
	Endpoint  string `json:"endpoint"`
	PublicKey string `json:"publicKey"`
}

// Load reads FileName from dir, or path if it is a file.
func Load(path string) (Config, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Config{}, err
	}
	if info.IsDir() {
		path = filepath.Join(path, FileName)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c, err := Parse(b)
	if err != nil {
		return c, err
	}
	// The icon is named relative to the manifest, the way main is, not
	// relative to wherever the command happened to be run from. `app build`
	// takes a directory argument and joins it onto main and the output dir
	// already; leaving the icon out meant building a project from anywhere
	// but its own root failed on the icon alone.
	if c.Icon != "" && !filepath.IsAbs(c.Icon) {
		c.Icon = filepath.Join(filepath.Dir(path), c.Icon)
	}
	return c, nil
}

// Parse decodes a Config, filling defaults.
func Parse(b []byte) (Config, error) {
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("appmeta: %w", err)
	}
	c.fill()
	if err := c.validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c *Config) fill() {
	if c.Version == "" {
		c.Version = "0.0.0"
	}
	if c.Main == "" {
		c.Main = "."
	}
	if c.Identifier == "" && c.Name != "" {
		c.Identifier = "com.example." + slug(c.Name)
	}
}

func (c Config) validate() error {
	if c.Name == "" {
		return fmt.Errorf("appmeta: name is required")
	}
	if c.Identifier == "" {
		return fmt.Errorf("appmeta: identifier is required")
	}
	if strings.ContainsAny(c.Identifier, " \t") {
		return fmt.Errorf("appmeta: identifier %q must not contain spaces", c.Identifier)
	}
	// These strings are interpolated raw into line-oriented bundle files — a
	// .desktop entry (Exec=), nfpm YAML (a postinstall hook), an Info.plist.
	// A newline or other control character in any of them would inject a new
	// directive that runs when a victim installs or launches the app, so the
	// config is refused at the door rather than escaped at each emitter.
	fields := []struct{ name, val string }{
		{"name", c.Name},
		{"identifier", c.Identifier},
		{"version", c.Version},
		{"copyright", c.Copyright},
		{"category", c.Category},
		{"windows.fileDescription", c.Bundle.Windows.FileDescription},
		{"linux.desktop", c.Bundle.Linux.Desktop},
	}
	for _, cat := range c.Bundle.Linux.Categories {
		fields = append(fields, struct{ name, val string }{"linux.categories entry", cat})
	}
	for _, f := range fields {
		if i := strings.IndexFunc(f.val, isControl); i >= 0 {
			return fmt.Errorf("appmeta: %s must not contain control characters (found one at byte %d)", f.name, i)
		}
	}
	// The scheme becomes an x-scheme-handler MIME type and a plist URL scheme;
	// keep it to the characters RFC 3986 allows in one.
	if c.DeepLinkScheme != "" && !validScheme(c.DeepLinkScheme) {
		return fmt.Errorf("appmeta: deepLinkScheme %q is not a valid URL scheme", c.DeepLinkScheme)
	}
	return nil
}

func isControl(r rune) bool { return r < 0x20 || r == 0x7f }

// validScheme reports whether s is a syntactically valid URL scheme: an ASCII
// letter followed by letters, digits, '+', '-' or '.'.
func validScheme(s string) bool {
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && (r >= '0' && r <= '9' || r == '+' || r == '-' || r == '.'):
		default:
			return false
		}
	}
	return s != ""
}

// WriteJSON writes the config as indented JSON.
func (c Config) WriteJSON(path string) error {
	c.fill()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func slug(s string) string {
	var b bytes.Buffer
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '-':
			if b.Len() > 0 {
				b.WriteByte('-')
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// BinaryName is the executable name inside a bundle.
func (c Config) BinaryName() string {
	s := slug(c.Name)
	if s == "" {
		return "app"
	}
	return s
}
