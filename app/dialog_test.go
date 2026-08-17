package app

import (
	"context"
	"errors"
	"runtime"
	"testing"
)

func TestSaveFileCanceled(t *testing.T) {
	orig := dialogExec
	t.Cleanup(func() { dialogExec = orig })
	dialogExec = func(context.Context, string, ...string) ([]byte, error) {
		return canceledOutput()
	}
	_, err := SaveFile(context.Background(), FileFilter{Name: "JSON", Ext: ".json"})
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("SaveFile canceled = %v, want ErrCanceled", err)
	}
}

func TestOpenFileReturnsPath(t *testing.T) {
	orig := dialogExec
	t.Cleanup(func() { dialogExec = orig })
	dialogExec = func(context.Context, string, ...string) ([]byte, error) {
		return []byte("/tmp/board.json\n"), nil
	}
	path, err := OpenFile(context.Background(), FileDialog{})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tmp/board.json" {
		t.Fatalf("path = %q", path)
	}
}

func TestConfirmNo(t *testing.T) {
	orig := dialogExec
	t.Cleanup(func() { dialogExec = orig })
	dialogExec = func(context.Context, string, ...string) ([]byte, error) {
		return canceledOutput()
	}
	ok, err := Confirm(context.Background(), "Q", "Sure?")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false")
	}
}

func TestParseKeys(t *testing.T) {
	t.Parallel()
	k := parseKeys("CmdOrCtrl+Shift+E")
	if !k.ctrlOrCmd || !k.shift || k.key != "E" {
		t.Fatalf("parseKeys: %+v", k)
	}
}

func TestDefaultMenuHasQuit(t *testing.T) {
	t.Parallel()
	m := DefaultMenu()
	if len(m.Items) < 2 {
		t.Fatal("DefaultMenu too small")
	}
	found := false
	for _, it := range m.Items[0].Items {
		if it.Role == RoleQuit {
			found = true
		}
	}
	if !found {
		t.Fatal("File menu missing Quit")
	}
}

func canceledOutput() ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		return []byte("execution error: User canceled. (-128)"), errors.New("exit status 1")
	case "windows":
		return nil, &exitErr{code: 2, msg: "exit status 2"}
	default:
		return nil, &exitErr{code: 1, msg: "exit status 1"}
	}
}

type exitErr struct {
	code int
	msg  string
}

func (e *exitErr) Error() string { return e.msg }
func (e *exitErr) ExitCode() int { return e.code }
