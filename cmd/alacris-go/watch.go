package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func appDev(args []string, stdout, stderr io.Writer) error {
	watch := true
	dir := "."
	progArgs := args
	var filtered []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-watch=false", "-watch=0":
			watch = false
		case "-watch":
			watch = true
		default:
			filtered = append(filtered, args[i])
		}
	}
	progArgs = filtered
	if len(progArgs) > 0 && progArgs[0] != "--" && !flagish(progArgs[0]) {
		dir = progArgs[0]
		progArgs = progArgs[1:]
	}
	if len(progArgs) > 0 && progArgs[0] == "--" {
		progArgs = progArgs[1:]
	}
	if !watch {
		return runDesktop(dir, progArgs, stdout, stderr)
	}
	return watchDesktop(dir, progArgs, stdout, stderr)
}

func runDesktop(dir string, progArgs []string, stdout, stderr io.Writer) error {
	runArgs := append([]string{"run", "-tags", "desktop", "."}, progArgs...)
	cmd := exec.Command("go", runArgs...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdoutWriter(stdout)
	cmd.Stderr = stderrWriter(stderr)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	return cmd.Run()
}

func watchDesktop(dir string, progArgs []string, stdout, stderr io.Writer) error {
	fmt.Fprintln(stderrWriter(stderr), "alacris-go: app dev watching for changes (Ctrl+C to stop)")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	var cmd *exec.Cmd
	start := func() error {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		runArgs := append([]string{"run", "-tags", "desktop", "."}, progArgs...)
		cmd = exec.Command("go", runArgs...)
		cmd.Dir = dir
		cmd.Stdin = os.Stdin
		cmd.Stdout = stdoutWriter(stdout)
		cmd.Stderr = stderrWriter(stderr)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
		return cmd.Start()
	}
	if err := start(); err != nil {
		return err
	}

	stamp, err := snapshot(dir)
	if err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	tick := time.NewTicker(400 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-sig:
			if cmd != nil && cmd.Process != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
			return nil
		case <-tick.C:
			next, err := snapshot(dir)
			if err != nil {
				continue
			}
			if next != stamp {
				stamp = next
				fmt.Fprintln(stderrWriter(stderr), "alacris-go: reload")
				if err := start(); err != nil {
					fmt.Fprintln(stderrWriter(stderr), "alacris-go:", err)
				}
			}
		}
	}
}

func snapshot(root string) (string, error) {
	var b strings.Builder
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "dist" || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !watchable(rel) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		fmt.Fprintf(&b, "%s:%d:%d\n", rel, info.Size(), info.ModTime().UnixNano())
		return nil
	})
	return b.String(), err
}

func watchable(rel string) bool {
	base := filepath.Base(rel)
	if strings.HasSuffix(base, "_templ.go") || strings.HasSuffix(base, "_gen.go") {
		return false
	}
	switch filepath.Ext(base) {
	case ".go", ".templ", ".js", ".css", ".html", ".json":
		return true
	}
	return false
}
