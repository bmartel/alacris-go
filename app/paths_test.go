package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDir(t *testing.T) {
	t.Parallel()
	if _, err := DataDir(""); err == nil {
		t.Fatal("empty identifier succeeded")
	}
	dir, err := DataDir("dev.alacris.test-datadir")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "dev.alacris.test-datadir" {
		t.Fatalf("dir = %s", dir)
	}
}
