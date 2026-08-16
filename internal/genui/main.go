// Command genui generates the ui package from the vendored @alacris/ui sources.
//
//	go run ./internal/genui         # write github.com/bmartel/alacris-go/ui
//	go run ./internal/genui -check  # fail if the wrappers are out of date
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmartel/alacris-go/gen"
)

const packageDoc = `Package ui is the typed Go surface of Alacris UI — sixty-eight Material
Design 3 components, a three-tier token system, and a theme engine.

Enable it on a page with alacris.Config{UI: true}. That loads every component
and applies Material defaults (seed #e8ad18, Google Sans Flex, scheme from
the OS). Config.Theme re-skins the whole page; every component follows
because they consume system tokens.

A component's shadow content is rendered in the browser. What these functions
produce is the element, its props and its light-DOM children.

Code in this package is generated from the vendored @alacris/ui sources.
Do not edit it; change the upstream component or regenerate.`

func main() {
	log.SetFlags(0)
	log.SetPrefix("genui: ")

	check := flag.Bool("check", false, "fail if the generated package is out of date")
	src := flag.String("src", filepath.Join("assets", "ui", "components"), "component sources")
	out := flag.String("o", "ui", "output directory")
	flag.Parse()

	m, err := gen.Load(*src)
	if err != nil {
		log.Fatal(err)
	}
	if len(m.Components) == 0 {
		log.Fatalf("no components in %s", *src)
	}

	files, err := gen.Generate(m, gen.Options{
		Package:     "ui",
		StripPrefix: "ui-",
		RelativeTo:  *out,
		Live:        true,
		PackageDoc:  packageDoc,
	})
	if err != nil {
		log.Fatal(err)
	}

	if *check {
		if err := checkOut(*out, files); err != nil {
			log.Fatal(err)
		}
		fmt.Println("genui: up to date")
		return
	}

	written, err := gen.WriteFiles(*out, files)
	if err != nil {
		log.Fatal(err)
	}
	stale, err := gen.Stale(*out, files)
	if err != nil {
		log.Fatal(err)
	}
	for _, path := range stale {
		if err := os.Remove(path); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("genui: %d component(s) -> %s\n", len(m.Components), *out)
	for _, path := range written {
		fmt.Printf("  wrote   %s\n", path)
	}
	for _, path := range stale {
		fmt.Printf("  removed %s\n", path)
	}
	if len(written) == 0 && len(stale) == 0 {
		fmt.Println("  up to date")
	}
}

func checkOut(dir string, files []gen.File) error {
	var problems []string
	for _, f := range files {
		path := filepath.Join(dir, f.Name)
		old, err := os.ReadFile(path)
		switch {
		case os.IsNotExist(err):
			problems = append(problems, "missing: "+path)
		case err != nil:
			return err
		case string(old) != string(f.Source):
			problems = append(problems, "out of date: "+path)
		}
	}
	stale, err := gen.Stale(dir, files)
	if err != nil {
		return err
	}
	for _, path := range stale {
		problems = append(problems, "left over: "+path)
	}
	if len(problems) > 0 {
		return fmt.Errorf("the ui package does not match the vendored components:\n  %s\n\nrun: go run ./internal/genui",
			strings.Join(problems, "\n  "))
	}
	return nil
}
