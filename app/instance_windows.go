//go:build windows

package app

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	lockfileFailImmediately = 0x00000001
	lockfileExclusiveLock   = 0x00000002
)

// lockByteHigh is the OffsetHigh of the single byte the lock covers: 4 GiB in,
// far past the port and secret the file actually holds. Unlike flock, which is
// advisory and does not affect reads, a Windows exclusive lock is mandatory and
// blocks other processes from reading the locked range — so locking byte 0
// stopped a second instance from reading the handover port back out of the
// file. Locking a byte beyond the content keeps mutual exclusion (every
// instance contends for the same byte) while leaving the data readable.
const lockByteHigh = 1

func overlappedAtLockByte() syscall.Overlapped {
	return syscall.Overlapped{OffsetHigh: lockByteHigh}
}

func lockFile(f *os.File) error {
	ol := overlappedAtLockByte()
	r1, _, err := procLockFileEx.Call(
		f.Fd(),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&ol)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func unlockFile(f *os.File) {
	ol := overlappedAtLockByte()
	_, _, _ = procUnlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&ol)))
}
