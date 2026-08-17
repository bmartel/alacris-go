//go:build windows

package app

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func relaunchNow(exe string, args []string) error {
	pending := exe + ".new"
	if _, err := os.Stat(pending); err == nil {
		return relaunchReplaceWindows(exe, pending, args)
	}
	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func relaunchReplaceWindows(exe, pending string, args []string) error {
	quotedArgs := ""
	for _, a := range args {
		quotedArgs += " " + powershellQuote(a)
	}
	script := fmt.Sprintf(
		"Wait-Process -Id %s -ErrorAction SilentlyContinue; Move-Item -LiteralPath %s -Destination %s -Force; Start-Process -FilePath %s -ArgumentList @(%s)",
		strconv.Itoa(os.Getpid()),
		powershellQuote(pending),
		powershellQuote(exe),
		powershellQuote(exe),
		strings.TrimSpace(quotedArgs),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
