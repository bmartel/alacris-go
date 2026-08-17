package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func appInfo(_ []string, stdout, stderr io.Writer) error {
	out := stdoutWriter(stdout)
	fmt.Fprintf(out, "go:      %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(out, "cgo:     %s\n", os.Getenv("CGO_ENABLED"))
	if p, err := exec.LookPath("go"); err == nil {
		fmt.Fprintf(out, "go bin:  %s\n", p)
	}
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("xcodebuild"); err == nil {
			fmt.Fprintln(out, "xcode:   yes")
		} else {
			fmt.Fprintln(out, "xcode:   no (need Xcode Command Line Tools)")
		}
	case "linux":
		if _, err := exec.LookPath("pkg-config"); err != nil {
			fmt.Fprintln(out, "webkit:  pkg-config not found")
			break
		}
		for _, pc := range []string{"webkit2gtk-4.1", "webkit2gtk-4.0"} {
			cmd := exec.Command("pkg-config", "--exists", pc)
			if cmd.Run() == nil {
				fmt.Fprintf(out, "webkit:  %s\n", pc)
				break
			}
		}
	case "windows":
		fmt.Fprintln(out, "webview: WebView2 (Edge)")
	}
	if p, err := exec.LookPath("codesign"); err == nil {
		fmt.Fprintf(out, "codesign: %s\n", p)
	}
	if p, err := exec.LookPath("nfpm"); err == nil {
		fmt.Fprintf(out, "nfpm:    %s\n", p)
	}
	if p, err := exec.LookPath("go-winres"); err == nil {
		fmt.Fprintf(out, "winres:  %s\n", p)
	}
	_ = stderr
	return nil
}
