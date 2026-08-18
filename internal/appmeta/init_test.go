package appmeta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitWritesScaffold(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := Init(InitOptions{
		Dir:        dir,
		Name:       "Board",
		Identifier: "com.example.board",
		Module:     "example.com/board",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{FileName, "go.mod", "main.go", "page.templ", "web/components.js"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mod), "example.com/board") {
		t.Fatalf("go.mod:\n%s", mod)
	}
	if strings.Contains(string(mod), "v0.0.0") {
		t.Fatalf("go.mod must not pin v0.0.0:\n%s", mod)
	}
	main, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(main), "app.Run") {
		t.Fatal("main.go missing app.Run")
	}
	if !strings.Contains(string(main), "live.SecureNever") {
		t.Fatal("main.go should set CookieSecure: SecureNever")
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	opts := InitOptions{Dir: dir, Name: "X", Identifier: "com.example.x", Module: "x"}
	if err := Init(opts); err != nil {
		t.Fatal(err)
	}
	if err := Init(opts); err == nil {
		t.Fatal("second Init succeeded")
	}
}

func TestInitPinsReleasedVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := Init(InitOptions{
		Dir:            dir,
		Name:           "Board",
		Identifier:     "com.example.board",
		Module:         "example.com/board",
		AlacrisVersion: "v0.7.2",
	})
	if err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(mod)
	if !strings.Contains(s, "github.com/bmartel/alacris-go v0.7.2") {
		t.Fatalf("go.mod missing pinned alacris-go:\n%s", s)
	}
	if !strings.Contains(s, "github.com/bmartel/alacris-go/app v0.7.2") {
		t.Fatalf("go.mod missing pinned app module:\n%s", s)
	}
	if strings.Contains(s, "v0.0.0") {
		t.Fatalf("go.mod still pins v0.0.0:\n%s", s)
	}
}

func TestInitDropsZeroVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := Init(InitOptions{
		Dir:            dir,
		Name:           "Board",
		Identifier:     "com.example.board",
		Module:         "example.com/board",
		AlacrisVersion: "v0.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mod), "v0.0.0") {
		t.Fatalf("v0.0.0 should be treated as unset:\n%s", mod)
	}
}
