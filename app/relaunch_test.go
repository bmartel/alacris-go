package app

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestRelaunchCallsHook(t *testing.T) {
	orig := relaunchFn
	t.Cleanup(func() { relaunchFn = orig })
	called := false
	relaunchFn = func(exe string, args []string) error {
		called = true
		if exe == "" {
			t.Fatal("empty exe")
		}
		return nil
	}
	if err := Relaunch(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("relaunch hook not called")
	}
}

func TestApplyAndRelaunch(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	artifact := []byte("new-bytes")
	sig := SignArtifact(priv, artifact)
	dir := t.TempDir()
	target := filepath.Join(dir, "app")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	orig := relaunchFn
	t.Cleanup(func() { relaunchFn = orig })
	relaunchFn = func(string, []string) error { return nil }

	u := Updater{PublicKey: pub, ApplyTo: target}
	if err := u.ApplyAndRelaunch(context.Background(), &Update{
		URL:       "unused",
		Signature: sig,
	}); err == nil {
		t.Fatal("ApplyAndRelaunch without a download URL should fail on Apply")
	}

	// Apply then Relaunch, skipping the HTTP round-trip.
	if err := replaceFile(target, artifact); err != nil {
		t.Fatal(err)
	}
	if err := Relaunch(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(artifact) {
		t.Fatalf("got %q", got)
	}
}

func TestFinishPendingUpdateNoFile(t *testing.T) {
	if err := FinishPendingUpdate(); err != nil {
		t.Fatal(err)
	}
}
