//go:build unix

package app

import (
	"os"
	"syscall"
)

func relaunchNow(exe string, args []string) error {
	argv := append([]string{exe}, args...)
	return syscall.Exec(exe, argv, os.Environ())
}
