package appmeta

import (
	"strings"
	"testing"
)

func TestParseFillsDefaults(t *testing.T) {
	t.Parallel()
	c, err := Parse([]byte(`{"name":"Board"}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Identifier != "com.example.board" {
		t.Errorf("identifier = %q", c.Identifier)
	}
	if c.Version != "0.0.0" {
		t.Errorf("version = %q", c.Version)
	}
	if c.Main != "." {
		t.Errorf("main = %q", c.Main)
	}
}

func TestParseRejectsEmptyName(t *testing.T) {
	t.Parallel()
	if _, err := Parse([]byte(`{"identifier":"com.x"}`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRejectsSpacedIdentifier(t *testing.T) {
	t.Parallel()
	if _, err := Parse([]byte(`{"name":"X","identifier":"com x"}`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestSlug(t *testing.T) {
	t.Parallel()
	if g, w := slug("Ship It"), "ship-it"; g != w {
		t.Errorf("slug = %q, want %q", g, w)
	}
}

func TestInfoPlistContainsIdentifier(t *testing.T) {
	t.Parallel()
	c := Config{Name: "Board", Identifier: "com.example.board", Version: "1.2.3"}
	plist := infoPlist(c)
	for _, s := range []string{"com.example.board", "Board", "1.2.3", "CFBundleExecutable"} {
		if !strings.Contains(plist, s) {
			t.Errorf("Info.plist missing %q\n%s", s, plist)
		}
	}
}

func TestDesktopFile(t *testing.T) {
	t.Parallel()
	c := Config{Name: "Board", Identifier: "com.example.board"}
	d := desktopFile(c)
	if !strings.Contains(d, "Name=Board") || !strings.Contains(d, "Exec=") {
		t.Fatalf("desktop file:\n%s", d)
	}
}
