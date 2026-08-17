// Command alacris-go generates typed Go wrappers for alacris web components
// and hosts a desktop app around the same live handler.
//
//	alacris-go generate ./web/components -o ./internal/components
//	alacris-go manifest ./web/components -o ./alacris.components.json
//	alacris-go check    ./web/components -o ./internal/components
//	alacris-go app init|dev|build
//
// The usual place to put it is a go:generate line next to the templ one:
//
//	//go:generate go run github.com/bmartel/alacris-go/cmd/alacris-go generate ./web/components -o ./internal/components
//	//go:generate templ generate
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmartel/alacris-go/gen"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "alacris-go:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stderr)
		return flag.ErrHelp
	}

	switch args[0] {
	case "generate":
		return generate(args[1:], stdout, stderr, false)
	case "check":
		return generate(args[1:], stdout, stderr, true)
	case "manifest":
		return manifest(args[1:], stdout, stderr)
	case "app":
		return appCmd(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return nil
	case "version":
		fmt.Fprintln(stdout, "alacris-go", version())
		return nil
	default:
		fmt.Fprintf(stderr, "alacris-go: unknown command %q\n\n", args[0])
		usage(stderr)
		return flag.ErrHelp
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `alacris-go turns alacris component definitions into typed Go wrappers.

Usage:
  alacris-go generate <path>... -o <dir>   write Go wrappers
  alacris-go check    <path>... -o <dir>   fail if the wrappers are out of date
  alacris-go manifest <path>...            write what it found, as JSON
  alacris-go app init|dev|build            desktop host, around the same live app
  alacris-go version

Each <path> is a JavaScript file, a directory to walk, or a manifest .json
file. Run "alacris-go generate -h" for the flags.
`)
}

type commonFlags struct {
	out      string
	pkg      string
	strip    string
	optional string
	imp      string
	live     bool
}

func (c *commonFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&c.out, "o", "", "directory to write the generated package to (required)")
	fs.StringVar(&c.pkg, "package", "", "package name (default: the name of the output directory)")
	fs.StringVar(&c.strip, "strip", "", "tag name prefix to drop when deriving Go names, e.g. \"ala-\"")
	fs.StringVar(&c.optional, "optional", string(gen.OptionalZero),
		"how props with a non-zero default are represented: zero or pointer")
	fs.StringVar(&c.imp, "import", gen.DefaultImport, "import path of the alacris runtime package")
	fs.BoolVar(&c.live, "live", true,
		"also generate typed live patch handles (imports the live package; -live=false to omit)")
}

func (c *commonFlags) options() (gen.Options, error) {
	if c.out == "" {
		return gen.Options{}, fmt.Errorf("-o is required")
	}
	pkg := c.pkg
	if pkg == "" {
		abs, err := filepath.Abs(c.out)
		if err != nil {
			return gen.Options{}, err
		}
		pkg = packageName(filepath.Base(abs))
		if pkg == "" {
			return gen.Options{}, fmt.Errorf("cannot derive a package name from %q; pass -package", c.out)
		}
	}
	return gen.Options{
		Package:     pkg,
		Import:      c.imp,
		Optional:    gen.Optional(c.optional),
		StripPrefix: c.strip,
		RelativeTo:  c.out,
		Live:        c.live,
	}, nil
}

// parseFlags parses args, allowing flags and paths to be interleaved.
//
// The flag package stops at the first non-flag argument, which would make
// "alacris-go generate ./web -o ./ui" — the order the usage text asks for and
// the order everyone types — silently lose -o.
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// packageName turns a directory name into a legal Go package name.
func packageName(dir string) string {
	var b strings.Builder
	for _, r := range dir {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9' && b.Len() > 0, r == '_':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		}
	}
	return b.String()
}

func generate(args []string, stdout, stderr io.Writer, checkOnly bool) error {
	name := "generate"
	if checkOnly {
		name = "check"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)

	var common commonFlags
	common.register(fs)
	quiet := fs.Bool("q", false, "only report problems")
	writeManifest := fs.String("manifest", "", "also write the manifest to this file")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: alacris-go %s <path>... -o <dir>\n\n", name)
		if checkOnly {
			fmt.Fprintf(stderr, "Reports whether the generated package is up to date, and writes nothing.\nUse it in CI to catch a component change that was never regenerated.\n\n")
		}
		fs.PrintDefaults()
	}
	paths, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		fs.Usage()
		return fmt.Errorf("no component sources given")
	}

	opts, err := common.options()
	if err != nil {
		return err
	}

	m, err := gen.Load(paths...)
	if err != nil {
		return err
	}
	if len(m.Components) == 0 {
		return fmt.Errorf("no components found in %s", strings.Join(paths, ", "))
	}

	files, err := gen.Generate(m, opts)
	if err != nil {
		return err
	}

	if checkOnly {
		return check(common.out, files, stdout)
	}

	written, err := gen.WriteFiles(common.out, files)
	if err != nil {
		return err
	}
	stale, err := gen.Stale(common.out, files)
	if err != nil {
		return err
	}
	for _, path := range stale {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	if *writeManifest != "" {
		if err := writeManifestFile(*writeManifest, m); err != nil {
			return err
		}
	}

	if !*quiet {
		fmt.Fprintf(stdout, "alacris-go: %d component(s) -> %s\n", len(m.Components), common.out)
		for _, path := range written {
			fmt.Fprintf(stdout, "  wrote   %s\n", path)
		}
		for _, path := range stale {
			fmt.Fprintf(stdout, "  removed %s\n", path)
		}
		if len(written) == 0 && len(stale) == 0 {
			fmt.Fprintln(stdout, "  up to date")
		}
	}
	return nil
}

// check reports what generate would change, without changing it.
func check(dir string, files []gen.File, stdout io.Writer) error {
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
		return fmt.Errorf("the generated package does not match the components:\n  %s\n\nrun: alacris-go generate ... -o %s",
			strings.Join(problems, "\n  "), dir)
	}
	fmt.Fprintln(stdout, "alacris-go: up to date")
	return nil
}

func manifest(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("manifest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("o", "", "file to write to (default: standard output)")
	fs.Usage = func() {
		fmt.Fprint(stderr, `Usage: alacris-go manifest <path>... [-o file]

Writes what the scanner found, as JSON. Commit it and generate from it when a
component declares something the scanner cannot see from its define() call.

`)
		fs.PrintDefaults()
	}
	paths, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		fs.Usage()
		return fmt.Errorf("no component sources given")
	}

	m, err := gen.Load(paths...)
	if err != nil {
		return err
	}
	if err := m.Validate(); err != nil {
		return err
	}
	if *out == "" {
		return gen.WriteManifest(stdout, m)
	}
	return writeManifestFile(*out, m)
}

func writeManifestFile(path string, m *gen.Manifest) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gen.WriteManifest(f, m)
}
