package app

import (
	"context"
	"os"
	"path/filepath"
)

// ApplyAndRelaunch is Apply followed by Relaunch. On success this process
// is replaced or exited.
func (u Updater) ApplyAndRelaunch(ctx context.Context, upd *Update) error {
	if err := u.Apply(ctx, upd); err != nil {
		return err
	}
	return Relaunch()
}

// Relaunch starts a new copy of this executable with the same arguments
// and exits. On Windows, if a `.new` file is parked next to the running
// exe (Apply could not replace a locked file), that file is swapped in
// first.
func Relaunch() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		exe, _ = os.Executable()
	}
	return relaunchFn(exe, os.Args[1:])
}

// FinishPendingUpdate applies a parked `exe.new` from a previous Apply
// that could not replace a locked Windows binary. Run calls it. It is
// safe to call at the start of main.
func FinishPendingUpdate() error {
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	pending := exe + ".new"
	if _, err := os.Stat(pending); err != nil {
		return nil
	}
	if err := os.Rename(pending, exe); err != nil {
		// The running file is locked (typical on Windows). Relaunch
		// swaps .new in after this process exits.
		return nil
	}
	return nil
}

var relaunchFn = relaunchNow
