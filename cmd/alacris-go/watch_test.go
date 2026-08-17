package main

import "testing"

func TestWatchable(t *testing.T) {
	t.Parallel()
	if !watchable("main.go") || !watchable("page.templ") || !watchable("web/x.js") {
		t.Fatal("source files should be watched")
	}
	if watchable("page_templ.go") || watchable("button_gen.go") || watchable("README.md") {
		t.Fatal("generated and unrelated files should not be watched")
	}
}
