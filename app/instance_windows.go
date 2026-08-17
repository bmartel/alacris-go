//go:build windows

package app

import (
	"os"
	"syscall"
)

func lockFile(f *os.File) error {
	ol := new(syscall.Overlapped)
	return syscall.LockFileEx(syscall.Handle(f.Fd()),
		syscall.LOCKFILE_EXCLUSIVE_LOCK|syscall.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, ol)
}

func unlockFile(f *os.File) {
	ol := new(syscall.Overlapped)
	_ = syscall.UnlockFileEx(syscall.Handle(f.Fd()), 0, 1, 0, ol)
}
