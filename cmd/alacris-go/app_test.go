package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmartel/alacris-go/internal/appmeta"
)

func TestAppInitCommand(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	err := run([]string{"app", "init", "-name", "Board", "-id", "com.example.board", "-module", "example.com/board", dir}, stdout, stderr)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, stderr.Bytes())
	}
	if _, err := os.Stat(filepath.Join(dir, appmeta.FileName)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "wrote a desktop app") {
		t.Fatalf("stdout: %s", stdout.String())
	}
}

func TestAppUnknownCommand(t *testing.T) {
	err := run([]string{"app", "nope"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
}
