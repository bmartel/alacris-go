package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// The site's examples are committed as JSON so the docs build needs no Go
// toolchain. That copy is only trustworthy if something checks it, and `go
// test ./...` is the thing every contributor already runs.
func TestDocExamplesAreCurrent(t *testing.T) {
	root := filepath.Join("..", "..")
	status, err := generate(
		filepath.Join(root, "internal", "docsgen", "examples.go"),
		filepath.Join(root, "docs", "src", "generated", "examples.json"),
		true,
	)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Log(status)
}

func TestPrettyHTML(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"short element stays on one line": {
			`<x-y a="1"></x-y>`,
			`<x-y a="1"></x-y>`,
		},
		"short text stays inline": {
			`<x-y>hello</x-y>`,
			`<x-y>hello</x-y>`,
		},
		"children are indented": {
			`<x-y><div slot="a">one</div><div slot="b">two</div></x-y>`,
			"<x-y>\n  <div slot=\"a\">one</div>\n  <div slot=\"b\">two</div>\n</x-y>",
		},
		"a long start tag breaks its attributes": {
			`<x-y alpha="aaaaaaaaaaaaaaa" beta="bbbbbbbbbbbbbbbb" gamma="ccccccccccccccc"></x-y>`,
			"<x-y\n  alpha=\"aaaaaaaaaaaaaaa\"\n  beta=\"bbbbbbbbbbbbbbbb\"\n  gamma=\"ccccccccccccccc\"\n></x-y>",
		},
		"a bare attribute survives": {
			`<x-y hidden></x-y>`,
			`<x-y hidden></x-y>`,
		},
		"a void element gets no end tag": {
			`<meta charset="utf-8">`,
			`<meta charset="utf-8">`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := prettyHTML(c.in); got != c.want {
				t.Errorf("\n got:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
}

// A tag with a binding-heavy attribute must not lose any of it: the formatter
// only ever adds whitespace between attributes.
func TestPrettyHTMLPreservesContent(t *testing.T) {
	in := `<ala-todo-list items="[{&#34;id&#34;:1}]" data-ala-on="add:add-todo toggle:t"></ala-todo-list>`
	got := prettyHTML(in)
	for _, want := range []string{`items="[{&#34;id&#34;:1}]"`, `data-ala-on="add:add-todo toggle:t"`, "ala-todo-list"} {
		if !strings.Contains(got, want) {
			t.Errorf("formatting dropped %s:\n%s", want, got)
		}
	}
}
