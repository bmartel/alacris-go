package app

import (
	"context"
	"os/exec"
)

// FileFilter names one extension the file dialog should offer.
type FileFilter struct {
	Name string // "JSON"
	Ext  string // ".json"
}

// A FileDialog configures OpenFile or SaveFile.
type FileDialog struct {
	Title    string
	Filename string
	Filters  []FileFilter
}

// OpenFile asks the user to pick an existing file. The zero FileDialog
// offers every file. ErrCanceled means the user dismissed the dialog.
func OpenFile(ctx context.Context, dlg FileDialog) (string, error) {
	return openFile(ctx, dlg)
}

// SaveFile asks the user where to write a file.
func SaveFile(ctx context.Context, filters ...FileFilter) (string, error) {
	return saveFile(ctx, FileDialog{Filters: filters})
}

// SaveAs is SaveFile with a title and default filename.
func SaveAs(ctx context.Context, dlg FileDialog) (string, error) {
	return saveFile(ctx, dlg)
}

// Message shows an informational dialog with an OK button.
func Message(ctx context.Context, title, text string) error {
	return message(ctx, title, text)
}

// Confirm asks a yes/no question. false, nil means the user chose No.
func Confirm(ctx context.Context, title, text string) (bool, error) {
	return confirm(ctx, title, text)
}

// dialogExec runs a helper binary. Tests replace it.
var dialogExec = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}
