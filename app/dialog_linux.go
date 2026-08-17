//go:build linux

package app

import (
	"context"
	"fmt"
	"strings"
)

func openFile(ctx context.Context, dlg FileDialog) (string, error) {
	args := []string{"--file-selection"}
	if dlg.Title != "" {
		args = append(args, "--title", dlg.Title)
	}
	args = append(args, zenityFilters(dlg.Filters)...)
	return zenityPath(ctx, args)
}

func saveFile(ctx context.Context, dlg FileDialog) (string, error) {
	args := []string{"--file-selection", "--save", "--confirm-overwrite"}
	if dlg.Title != "" {
		args = append(args, "--title", dlg.Title)
	}
	name := dlg.Filename
	if name == "" && len(dlg.Filters) > 0 {
		name = "untitled" + dlg.Filters[0].Ext
	}
	if name != "" {
		args = append(args, "--filename", name)
	}
	args = append(args, zenityFilters(dlg.Filters)...)
	return zenityPath(ctx, args)
}

func message(ctx context.Context, title, text string) error {
	args := []string{"--info", "--text", text}
	if title != "" {
		args = append(args, "--title", title)
	}
	_, err := zenity(ctx, args)
	if err != nil && isExit1(err) {
		return nil
	}
	return err
}

func confirm(ctx context.Context, title, text string) (bool, error) {
	args := []string{"--question", "--text", text}
	if title != "" {
		args = append(args, "--title", title)
	}
	_, err := zenity(ctx, args)
	if err != nil {
		if isExit1(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func zenityPath(ctx context.Context, args []string) (string, error) {
	out, err := zenity(ctx, args)
	if err != nil {
		if isExit1(err) {
			return "", ErrCanceled
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func zenity(ctx context.Context, args []string) ([]byte, error) {
	out, err := dialogExec(ctx, "zenity", args...)
	if err == nil {
		return out, nil
	}
	if !isNotFound(err) {
		return out, err
	}
	out, err = dialogExec(ctx, "kdialog", kdialogArgs(args)...)
	if err != nil && isNotFound(err) {
		return nil, fmt.Errorf("app: no file dialog helper (install zenity or kdialog): %w", err)
	}
	return out, err
}

func zenityFilters(filters []FileFilter) []string {
	var args []string
	for _, f := range filters {
		ext := strings.TrimPrefix(f.Ext, ".")
		name := f.Name
		if name == "" {
			name = ext
		}
		args = append(args, "--file-filter", fmt.Sprintf("%s | *.%s", name, ext))
	}
	return args
}

func kdialogArgs(zenity []string) []string {
	// Best-effort: the caller already built zenity flags. kdialog is only
	// the fallback when zenity is missing; keep the first mode flag.
	for _, a := range zenity {
		switch a {
		case "--file-selection":
			return []string{"--getopenfilename"}
		case "--save":
			return []string{"--getsavefilename"}
		case "--info":
			return []string{"--msgbox", lastText(zenity)}
		case "--question":
			return []string{"--yesno", lastText(zenity)}
		}
	}
	return zenity
}

func lastText(args []string) string {
	for i, a := range args {
		if a == "--text" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func isExit1(err error) bool {
	type exiter interface{ ExitCode() int }
	if e, ok := err.(exiter); ok {
		return e.ExitCode() == 1
	}
	return strings.Contains(err.Error(), "exit status 1")
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "executable file not found")
}
