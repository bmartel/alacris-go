//go:build desktop && windows

package app

/*
#cgo LDFLAGS: -lshell32 -luser32
#include <stdlib.h>
void appTrayInstall(void *hwnd, const char *tip);
void appTrayAddItem(int id, const char *title);
void appTrayAddSeparator(void);
void appTrayFinish(void);
*/
import "C"
import "unsafe"

func applyTray(w *Window, t *Tray) {
	if t == nil || w == nil {
		return
	}
	hwnd := unsafe.Pointer(w.handle())
	if hwnd == nil {
		return
	}
	tip := t.Tooltip
	if tip == "" {
		tip = t.Title
	}
	cs := C.CString(tip)
	C.appTrayInstall(hwnd, cs)
	C.free(unsafe.Pointer(cs))
	if t.Menu != nil {
		for _, it := range t.Menu.Items {
			addWinTrayItem(w, it)
		}
	}
	C.appTrayFinish()
}

func addWinTrayItem(w *Window, it MenuItem) {
	if it.Role == RoleSeparator {
		C.appTrayAddSeparator()
		return
	}
	do := it.Do
	if it.Role == RoleQuit {
		do = func(win *Window) { win.Close() }
	}
	if do == nil {
		return
	}
	id := registerMenuDo(w, do)
	title := C.CString(orTitle(it.Title, roleTitle(it.Role)))
	C.appTrayAddItem(C.int(id), title)
	C.free(unsafe.Pointer(title))
}
