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
	if !strings.Contains(string(mod), "github.com/bmartel/alacris-go/app") {
		t.Fatal("go.mod missing app module")
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
