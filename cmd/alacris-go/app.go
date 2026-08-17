package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/bmartel/alacris-go/internal/appmeta"
)

func appCmd(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		appUsage(stderr)
		return flag.ErrHelp
	}
	switch args[0] {
	case "init":
		return appInit(args[1:], stdout, stderr)
	case "dev":
		return appDev(args[1:], stdout, stderr)
	case "build":
		return appBuild(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		appUsage(stdout)
		return nil
	default:
		fmt.Fprintf(stderr, "alacris-go app: unknown command %q\n\n", args[0])
		appUsage(stderr)
		return flag.ErrHelp
	}
}

func appUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  alacris-go app init   [-name name] [-id identifier] [-module path] [dir]
  alacris-go app dev    [dir] [--] [args...]
  alacris-go app build  [-o dist] [-config alacris.app.json] [dir]

init writes a desktop-first project (Go, templ, alacris.app.json).
dev runs it with -tags desktop.
build compiles with -tags desktop and wraps the binary (.app, .desktop, .exe).
`)
}

func appInit(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("app init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("name", "", "window and bundle name (default: directory name)")
	id := fs.String("id", "", "reverse-DNS identifier (default: com.example.<name>)")
	mod := fs.String("module", "", "Go module path")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: alacris-go app init [dir]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	opts := appmeta.InitOptions{Dir: dir, Name: *name, Identifier: *id, Module: *mod}
	if err := appmeta.Init(opts); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "alacris-go: wrote a desktop app in %s\n", dir)
	fmt.Fprintf(stdout, "  next: cd %s && go get github.com/bmartel/alacris-go@latest github.com/bmartel/alacris-go/app@latest\n", dir)
	fmt.Fprintf(stdout, "        go generate ./... && alacris-go app dev\n")
	return nil
}

func appDev(args []string, stdout, stderr io.Writer) error {
	dir := "."
	progArgs := args
	if len(args) > 0 && args[0] != "--" && !flagish(args[0]) {
		dir = args[0]
		progArgs = args[1:]
	}
	if len(progArgs) > 0 && progArgs[0] == "--" {
		progArgs = progArgs[1:]
	}
	runArgs := append([]string{"run", "-tags", "desktop", "."}, progArgs...)
	cmd := exec.Command("go", runArgs...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdoutWriter(stdout)
	cmd.Stderr = stderrWriter(stderr)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	return cmd.Run()
}

func appBuild(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("app build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("o", "dist", "directory to write the bundle into")
	cfgPath := fs.String("config", "", "path to alacris.app.json (default: <dir>/alacris.app.json)")
	pkg := fs.String("pkg", "", "Go package to build (default: config.main)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: alacris-go app build [dir]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	cfgFile := *cfgPath
	if cfgFile == "" {
		cfgFile = filepath.Join(dir, appmeta.FileName)
	}
	cfg, err := appmeta.Load(cfgFile)
	if err != nil {
		return err
	}
	mainPkg := *pkg
	if mainPkg == "" {
		mainPkg = cfg.Main
		if !filepath.IsAbs(mainPkg) {
			mainPkg = filepath.Join(dir, mainPkg)
		}
	}
	tmp, err := os.MkdirTemp("", "alacris-app-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	binName := cfg.BinaryName()
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	bin := filepath.Join(tmp, binName)
	cmd := exec.Command("go", "build", "-tags", "desktop", "-o", bin, mainPkg)
	cmd.Dir = dir
	cmd.Stdout = stdoutWriter(stdout)
	cmd.Stderr = stderrWriter(stderr)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build -tags desktop: %w", err)
	}
	dest := *out
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(dir, dest)
	}
	got, err := appmeta.BundleDir(cfg, bin, dest)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "alacris-go: wrote %s\n", got)
	return nil
}

func flagish(s string) bool { return len(s) > 1 && s[0] == '-' }

func stdoutWriter(w io.Writer) io.Writer {
	if w == nil {
		return os.Stdout
	}
	return w
}

func stderrWriter(w io.Writer) io.Writer {
	if w == nil {
		return os.Stderr
	}
	return w
}
