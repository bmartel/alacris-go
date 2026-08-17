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

// A newline in a metadata field would inject a directive into a line-oriented
// bundle file — a second Exec= into a .desktop entry, a postinstall hook into
// nfpm YAML — that runs when a victim installs or launches the app. Parse must
// refuse it rather than let an emitter paste it through.
func TestParseRejectsControlCharactersInBundleFields(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"name":      `{"name":"Evil\nExec=/bin/sh -c pwn","identifier":"com.x"}`,
		"copyright": `{"name":"X","identifier":"com.x","copyright":"c\npostinstall: pwn"}`,
		"version":   `{"name":"X","identifier":"com.x","version":"1.0\nurl: evil"}`,
		"category":  `{"name":"X","identifier":"com.x","bundle":{"linux":{"categories":["Utility\nExec=pwn"]}}}`,
	}
	for field, in := range cases {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: expected Parse to reject a control character", field)
		}
	}
}

func TestParseRejectsBadDeepLinkScheme(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"ev il", "ev/il", "1evil", "evil\n"} {
		in := `{"name":"X","identifier":"com.x","deepLinkScheme":"` + s + `"}`
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("expected Parse to reject deepLinkScheme %q", s)
		}
	}
	if _, err := Parse([]byte(`{"name":"X","identifier":"com.x","deepLinkScheme":"my-app.v2"}`)); err != nil {
		t.Errorf("a valid scheme was rejected: %v", err)
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
