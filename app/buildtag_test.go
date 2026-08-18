package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopTagHidesWebviewImport(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		src, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, e.Name(), src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		if !importsWebview(f) {
			continue
		}
		if f.Name.Name != "app" {
			continue
		}
		tags := buildTags(string(src))
		if !strings.Contains(tags, "desktop") {
			t.Errorf("%s imports webview_go without a desktop build tag", e.Name())
		}
	}
}

func importsWebview(f *ast.File) bool {
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == "github.com/webview/webview_go" {
			return true
		}
	}
	return false
}

func buildTags(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "//go:build") || strings.HasPrefix(line, "// +build") {
			b.WriteString(line)
			b.WriteByte(' ')
			continue
		}
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		break
	}
	return b.String()
}

func TestModulePath(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "module github.com/bmartel/alacris-go/app") {
		t.Fatal("app go.mod has the wrong module path")
	}
	root := filepath.Join("..", "go.mod")
	rb, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(rb), "\n") {
		trim := strings.TrimSpace(line)
		if strings.Contains(trim, "webview") && !strings.Contains(trim, "// indirect") {
			t.Fatal("root go.mod must not directly require a webview library; it belongs in the app module")
		}
		if strings.HasPrefix(trim, "require github.com/bmartel/alacris-go/app v0.0.0") &&
			!strings.Contains(trim, "v0.0.0-") {
			t.Fatal("root go.mod must require a resolvable app version, not v0.0.0")
		}
	}
}

func TestAppGoModPinsWebview(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "github.com/webview/webview_go") {
		t.Fatal("app go.mod must require webview_go (imported under -tags desktop)")
	}
}
