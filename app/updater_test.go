package app

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// The manifest is unsigned, so its authenticity rests on TLS. A plaintext
// endpoint (except loopback) is refused before any bytes are fetched, which is
// what keeps a network attacker from advertising a bumped version that points
// at an older validly-signed artifact.
func TestCheckRequiresHTTPS(t *testing.T) {
	t.Parallel()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	u := Updater{Endpoint: "http://updates.example.com/latest.json", PublicKey: pub, Current: "0.1.0"}
	if _, err := u.Check(context.Background()); err == nil {
		t.Fatal("Check accepted a plaintext non-loopback endpoint")
	}
}

func TestApplyRequiresHTTPS(t *testing.T) {
	t.Parallel()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	u := Updater{PublicKey: pub}
	upd := &Update{URL: "http://dl.example.com/app.tar.gz", Version: "9.9.9"}
	if err := u.Apply(context.Background(), upd); err == nil {
		t.Fatal("Apply accepted a plaintext non-loopback artifact URL")
	}
}

func TestVerifyRoundTrip(t *testing.T) {
	t.Parallel()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("the artifact")
	sig := SignArtifact(priv, body)
	if err := Verify(pub, body, sig); err != nil {
		t.Fatal(err)
	}
	if err := Verify(pub, []byte("tampered"), sig); err == nil {
		t.Fatal("tampered artifact verified")
	}
}

func TestNewer(t *testing.T) {
	t.Parallel()
	if !newer("0.2.0", "0.1.9") {
		t.Fatal("0.2.0 should be newer than 0.1.9")
	}
	if newer("0.1.0", "0.1.0") {
		t.Fatal("equal versions are not newer")
	}
	if newer("0.1.0", "0.2.0") {
		t.Fatal("older is not newer")
	}
	if !newer("1.0.0", "0.9.9") {
		t.Fatal("1.0.0 should be newer than 0.9.9")
	}
	if !newer("v0.3.0", "0.2.0") {
		t.Fatal("v-prefix should be ignored")
	}
}

func TestCheckAndApply(t *testing.T) {
	t.Parallel()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	artifact := []byte("new-binary-bytes")
	sig := SignArtifact(priv, artifact)

	mux := http.NewServeMux()
	mux.HandleFunc("/latest.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Manifest{
			Version: "0.2.0",
			Notes:   "fixes",
			Platforms: map[string]PlatformUpdate{
				PlatformKey(): {URL: "http://" + r.Host + "/artifact.bin", Signature: sig},
			},
		})
	})
	mux.HandleFunc("/artifact.bin", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(artifact)
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	target := filepath.Join(dir, "app")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := Updater{
		Endpoint:  ts.URL + "/latest.json",
		PublicKey: pub,
		Current:   "0.1.0",
		HTTP:      ts.Client(),
		ApplyTo:   target,
	}
	upd, err := u.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if upd == nil || upd.Version != "0.2.0" {
		t.Fatalf("update = %+v", upd)
	}
	if err := u.Apply(context.Background(), upd); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(artifact) {
		t.Fatalf("replaced bytes = %q", got)
	}
}

func TestCheckCurrentIsNil(t *testing.T) {
	t.Parallel()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	sig := SignArtifact(priv, []byte("x"))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Manifest{
			Version: "0.1.0",
			Platforms: map[string]PlatformUpdate{
				PlatformKey(): {URL: "http://example/x", Signature: sig},
			},
		})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	u := Updater{Endpoint: ts.URL, PublicKey: pub, Current: "0.1.0", HTTP: ts.Client()}
	upd, err := u.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if upd != nil {
		t.Fatalf("current version offered an update: %+v", upd)
	}
}
