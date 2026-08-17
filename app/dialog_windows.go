//go:build windows

package app

import (
	"context"
	"fmt"
	"strings"
)

func openFile(ctx context.Context, dlg FileDialog) (string, error) {
	ps := strings.Join([]string{
		`Add-Type -AssemblyName System.Windows.Forms;`,
		`$f = New-Object System.Windows.Forms.OpenFileDialog;`,
		filterPS("$f", dlg),
		titlePS("$f", dlg.Title),
		`if ($f.ShowDialog() -ne [System.Windows.Forms.DialogResult]::OK) { exit 2 };`,
		`[Console]::Out.Write($f.FileName)`,
	}, " ")
	return powershellPath(ctx, ps)
}

func saveFile(ctx context.Context, dlg FileDialog) (string, error) {
	name := dlg.Filename
	if name == "" && len(dlg.Filters) > 0 {
		name = "untitled" + dlg.Filters[0].Ext
	}
	ps := strings.Join([]string{
		`Add-Type -AssemblyName System.Windows.Forms;`,
		`$f = New-Object System.Windows.Forms.SaveFileDialog;`,
		filterPS("$f", dlg),
		titlePS("$f", dlg.Title),
		filenamePS("$f", name),
		`if ($f.ShowDialog() -ne [System.Windows.Forms.DialogResult]::OK) { exit 2 };`,
		`[Console]::Out.Write($f.FileName)`,
	}, " ")
	return powershellPath(ctx, ps)
}

func message(ctx context.Context, title, text string) error {
	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.MessageBox]::Show(%s, %s) | Out-Null`,
		psString(text), psString(title))
	_, err := dialogExec(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	return err
}

func confirm(ctx context.Context, title, text string) (bool, error) {
	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; $r = [System.Windows.Forms.MessageBox]::Show(%s, %s, 'YesNo'); if ($r -eq 'Yes') { exit 0 } else { exit 2 }`,
		psString(text), psString(title))
	_, err := dialogExec(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if err != nil {
		if isExit2(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func powershellPath(ctx context.Context, ps string) (string, error) {
	out, err := dialogExec(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if err != nil {
		if isExit2(err) {
			return "", ErrCanceled
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func filterPS(varName string, dlg FileDialog) string {
	if len(dlg.Filters) == 0 {
		return ""
	}
	var parts []string
	for _, f := range dlg.Filters {
		ext := strings.TrimPrefix(f.Ext, ".")
		name := f.Name
		if name == "" {
			name = ext
		}
		parts = append(parts, fmt.Sprintf("%s (*.%s)|*.%s", name, ext, ext))
	}
	return fmt.Sprintf(`%s.Filter = %s;`, varName, psString(strings.Join(parts, "|")))
}

func titlePS(varName, title string) string {
	if title == "" {
		return ""
	}
	return fmt.Sprintf(`%s.Title = %s;`, varName, psString(title))
}

func filenamePS(varName, name string) string {
	if name == "" {
		return ""
	}
	return fmt.Sprintf(`%s.FileName = %s;`, varName, psString(name))
}

func psString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func isExit2(err error) bool {
	type exiter interface{ ExitCode() int }
	if e, ok := err.(exiter); ok {
		return e.ExitCode() == 2
	}
	return strings.Contains(err.Error(), "exit status 2")
}
