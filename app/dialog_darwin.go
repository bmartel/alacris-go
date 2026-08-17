//go:build darwin

package app

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

func openFile(ctx context.Context, dlg FileDialog) (string, error) {
	script := "POSIX path of (choose file"
	if dlg.Title != "" {
		script += " with prompt " + osaString(dlg.Title)
	}
	script += ")"
	return osaPath(ctx, script)
}

func saveFile(ctx context.Context, dlg FileDialog) (string, error) {
	script := "POSIX path of (choose file name"
	if dlg.Title != "" {
		script += " with prompt " + osaString(dlg.Title)
	}
	name := dlg.Filename
	if name == "" && len(dlg.Filters) > 0 {
		name = "untitled" + dlg.Filters[0].Ext
	}
	if name != "" {
		script += " default name " + osaString(name)
	}
	script += ")"
	return osaPath(ctx, script)
}

func message(ctx context.Context, title, text string) error {
	_, err := osa(ctx, fmt.Sprintf("display dialog %s with title %s buttons {\"OK\"} default button \"OK\"",
		osaString(text), osaString(title)))
	return err
}

func confirm(ctx context.Context, title, text string) (bool, error) {
	out, err := osa(ctx, fmt.Sprintf("display dialog %s with title %s buttons {\"Cancel\", \"OK\"} default button \"OK\"",
		osaString(text), osaString(title)))
	if err != nil {
		if isCanceled(err, out) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func osaPath(ctx context.Context, script string) (string, error) {
	out, err := osa(ctx, script)
	if err != nil {
		if isCanceled(err, out) {
			return "", ErrCanceled
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func osa(ctx context.Context, script string) ([]byte, error) {
	out, err := dialogExec(ctx, "osascript", "-e", script)
	if err != nil {
		return out, err
	}
	return out, nil
}

func osaString(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func isCanceled(err error, out []byte) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error() + string(out))
	return bytes.Contains([]byte(msg), []byte("user canceled")) ||
		bytes.Contains([]byte(msg), []byte("-128"))
}
