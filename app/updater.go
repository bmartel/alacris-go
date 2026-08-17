package app

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Manifest is the JSON an updater endpoint serves. The shape matches
// Tauri's updater so a GitHub Release can host one file both tools read.
//
//	{
//	  "version": "0.2.0",
//	  "notes": "…",
//	  "pub_date": "2026-08-17T12:00:00Z",
//	  "platforms": {
//	    "darwin-arm64": {
//	      "url": "https://example/Board_0.2.0_darwin-arm64.tar.gz",
//	      "signature": "<base64 ed25519 of the artifact bytes>"
//	    }
//	  }
//	}
type Manifest struct {
	Version   string                    `json:"version"`
	Notes     string                    `json:"notes,omitempty"`
	PubDate   string                    `json:"pub_date,omitempty"`
	Platforms map[string]PlatformUpdate `json:"platforms"`
}

// PlatformUpdate is one OS/arch artifact.
type PlatformUpdate struct {
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

// PlatformKey is the map key for this binary: GOOS-GOARCH, e.g. darwin-arm64.
func PlatformKey() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

// An Update is a verified, newer build ready to download.
type Update struct {
	Version   string
	Notes     string
	URL       string
	Signature string
	Current   string
}

// Updater checks an HTTP endpoint for a newer signed artifact.
type Updater struct {
	// Endpoint is the URL of a Manifest JSON file.
	Endpoint string
	// PublicKey verifies PlatformUpdate.Signature. Required.
	PublicKey ed25519.PublicKey
	// Current is this binary's version. Compared as a dotted triple when
	// both sides look like semver; otherwise string inequality is enough
	// to offer the update.
	Current string
	// HTTP is the client used to fetch. Defaults to http.DefaultClient.
	HTTP *http.Client
	// ApplyTo is the path replaced on Apply. Empty means os.Executable.
	ApplyTo string
}

// Check fetches the manifest and reports whether a newer signed artifact
// exists for this platform. A nil Update means the running version is
// current. Signature is not verified until Apply downloads the bytes.
func (u Updater) Check(ctx context.Context) (*Update, error) {
	if u.Endpoint == "" {
		return nil, fmt.Errorf("app: Updater.Endpoint is required")
	}
	if len(u.PublicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("app: Updater.PublicKey is required")
	}
	client := u.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("app: updater endpoint %s: HTTP %s", u.Endpoint, res.Status)
	}
	var man Manifest
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&man); err != nil {
		return nil, fmt.Errorf("app: updater manifest: %w", err)
	}
	plat, ok := man.Platforms[PlatformKey()]
	if !ok {
		return nil, fmt.Errorf("app: no artifact for %s", PlatformKey())
	}
	if !newer(man.Version, u.Current) {
		return nil, nil
	}
	return &Update{
		Version:   man.Version,
		Notes:     man.Notes,
		URL:       plat.URL,
		Signature: plat.Signature,
		Current:   u.Current,
	}, nil
}

// Apply downloads the artifact, verifies the ed25519 signature, and
// replaces the running binary. The new file is used on the next start;
// this process is not restarted.
func (u Updater) Apply(ctx context.Context, upd *Update) error {
	if upd == nil {
		return fmt.Errorf("app: no update")
	}
	if len(u.PublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("app: Updater.PublicKey is required")
	}
	client := u.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upd.URL, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("app: download %s: HTTP %s", upd.URL, res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 512<<20))
	if err != nil {
		return err
	}
	if err := Verify(u.PublicKey, body, upd.Signature); err != nil {
		return err
	}
	target := u.ApplyTo
	if target == "" {
		target, err = os.Executable()
		if err != nil {
			return err
		}
	}
	return replaceFile(target, body)
}

// Verify reports whether sigB64 is an ed25519 signature of artifact under pub.
func Verify(pub ed25519.PublicKey, artifact []byte, sigB64 string) error {
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("app: public key length %d, want %d", len(pub), ed25519.PublicKeySize)
	}
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		sig, err = base64.RawURLEncoding.DecodeString(sigB64)
		if err != nil {
			return fmt.Errorf("app: signature: %w", err)
		}
	}
	if !ed25519.Verify(pub, artifact, sig) {
		return fmt.Errorf("app: update signature does not match")
	}
	return nil
}

// SignArtifact returns the base64 signature to put in a Manifest. The
// matching public key is what the app embeds.
func SignArtifact(priv ed25519.PrivateKey, artifact []byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(priv, artifact))
}

func replaceFile(target string, body []byte) error {
	dir := filepath.Dir(target)
	f, err := os.CreateTemp(dir, ".alacris-update-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(body); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Chmod(0o755); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		if runtime.GOOS == "windows" {
			// A running executable cannot be replaced. Park the new
			// bytes next to it for the next launch.
			dest := target + ".new"
			_ = os.Remove(dest)
			if err2 := os.Rename(tmp, dest); err2 == nil {
				return nil
			}
		}
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func newer(next, current string) bool {
	next, current = strings.TrimPrefix(next, "v"), strings.TrimPrefix(current, "v")
	if current == "" {
		return next != ""
	}
	if next == current {
		return false
	}
	np, nOK := parseTriple(next)
	cp, cOK := parseTriple(current)
	if nOK && cOK {
		for i := 0; i < 3; i++ {
			if np[i] != cp[i] {
				return np[i] > cp[i]
			}
		}
		return false
	}
	return next != current
}

func parseTriple(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.SplitN(v, ".", 4)
	if len(parts) < 2 {
		return out, false
	}
	for i := 0; i < 3 && i < len(parts); i++ {
		n := 0
		for _, r := range parts[i] {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out[i] = n
	}
	return out, true
}
