package gen

import (
	"strings"
	"testing"
)

func liveManifest() *Manifest {
	return &Manifest{
		Version: ManifestVersion,
		Components: []Component{{
			Tag: "todo-list",
			Props: []Prop{
				{Name: "items", Attr: "items", Kind: KindJSON, GoType: "[]string"},
				{Name: "maxCount", Attr: "max-count", Kind: KindNumber, GoType: "int"},
			},
		}},
	}
}

func TestLiveEmitsTypedHandles(t *testing.T) {
	files, err := Generate(liveManifest(), Options{Package: "ui", Live: true})
	if err != nil {
		t.Fatal(err)
	}
	src := string(files[0].Source)

	for _, want := range []string{
		`live "github.com/bmartel/alacris-go/live"`,
		"type TodoListHandle struct",
		"func TodoListElement(s *live.Session, id string) TodoListHandle",
		"func (h TodoListHandle) Handle() live.Handle",
		// The setter uses the JavaScript prop name, not the kebab attribute:
		// live patches write the DOM property define() declared.
		`func (h TodoListHandle) SetItems(v []string) { h.handle.Set("items", v) }`,
		`func (h TodoListHandle) SetMaxCount(v int) { h.handle.Set("maxCount", v) }`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated source is missing %q", want)
		}
	}
}

func TestWithoutLiveNoHandleAndNoImport(t *testing.T) {
	files, err := Generate(liveManifest(), Options{Package: "ui"})
	if err != nil {
		t.Fatal(err)
	}
	src := string(files[0].Source)
	for _, banned := range []string{"live.Handle", "alacris-go/live", "TodoListHandle"} {
		if strings.Contains(src, banned) {
			t.Errorf("without Live, generated source must not contain %q", banned)
		}
	}
}

func TestLiveImportFollowsImportOverride(t *testing.T) {
	files, err := Generate(liveManifest(), Options{
		Package: "ui",
		Import:  "example.test/fork/alacris",
		Live:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(files[0].Source), `live "example.test/fork/alacris/live"`) {
		t.Error("LiveImport did not follow the overridden Import path")
	}
}
