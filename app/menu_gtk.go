//go:build desktop && !windows && !darwin

package app

/*
#cgo pkg-config: gtk+-3.0
#include <stdlib.h>
void appGtkInstallMenu(void *win);
void appGtkAddMenuItem(int id, const char* title, const char* key, int shift, int alt, int ctrl);
void appGtkAddSeparator(void);
void appGtkBeginSubmenu(const char* title);
void appGtkEndSubmenu(void);
void appGtkFinishMenu(void);
*/
import "C"
import (
	"strings"
	"unsafe"
)

func applyMenu(w *Window, m *Menu) {
	if m == nil || len(m.Items) == 0 || w == nil {
		return
	}
	hwnd := unsafe.Pointer(w.handle())
	if hwnd == nil {
		return
	}
	resetMenuFns()
	C.appGtkInstallMenu(hwnd)
	for _, it := range m.Items {
		addGtkMenuItem(w, it)
	}
	C.appGtkFinishMenu()
}

func addGtkMenuItem(w *Window, it MenuItem) {
	if it.Role == RoleSeparator {
		C.appGtkAddSeparator()
		return
	}
	if len(it.Items) > 0 {
		title := C.CString(it.Title)
		C.appGtkBeginSubmenu(title)
		C.free(unsafe.Pointer(title))
		for _, child := range it.Items {
			addGtkMenuItem(w, child)
		}
		C.appGtkEndSubmenu()
		return
	}
	do := it.Do
	if it.Role == RoleQuit {
		do = func(win *Window) { win.Close() }
	}
	if do == nil && it.Role == RoleNone {
		return
	}
	if do == nil {
		do = func(*Window) {}
	}
	id := registerMenuDo(w, do)
	k := parseKeys(it.Keys)
	key := C.CString(strings.ToLower(k.key))
	title := C.CString(orTitle(it.Title, roleTitle(it.Role)))
	shift, alt, ctrl := C.int(0), C.int(0), C.int(0)
	if k.shift {
		shift = 1
	}
	if k.alt {
		alt = 1
	}
	if k.ctrlOrCmd {
		ctrl = 1
	}
	C.appGtkAddMenuItem(C.int(id), title, key, shift, alt, ctrl)
	C.free(unsafe.Pointer(title))
	C.free(unsafe.Pointer(key))
}
