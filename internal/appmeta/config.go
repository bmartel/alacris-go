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
	return Parse(b)
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
	return nil
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
