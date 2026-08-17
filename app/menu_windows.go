//go:build desktop && windows

package app

/*
#cgo LDFLAGS: -luser32
#include <stdlib.h>
void appWinInstallMenu(void *hwnd);
void appWinAddMenuItem(int id, const char* title, const char* key, int shift, int alt, int ctrl);
void appWinAddSeparator(void);
void appWinBeginSubmenu(const char* title);
void appWinEndSubmenu(void);
void appWinFinishMenu(void *hwnd);
void appWinAddRoleQuit(int id, const char* title);
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
	C.appWinInstallMenu(hwnd)
	for _, it := range m.Items {
		addWinMenuItem(w, it)
	}
	C.appWinFinishMenu(hwnd)
}

func addWinMenuItem(w *Window, it MenuItem) {
	if it.Role == RoleSeparator {
		C.appWinAddSeparator()
		return
	}
	if len(it.Items) > 0 {
		title := C.CString(it.Title)
		C.appWinBeginSubmenu(title)
		C.free(unsafe.Pointer(title))
		for _, child := range it.Items {
			addWinMenuItem(w, child)
		}
		C.appWinEndSubmenu()
		return
	}
	k := parseKeys(it.Keys)
	key := C.CString(strings.ToLower(k.key))
	defer C.free(unsafe.Pointer(key))

	if it.Role == RoleQuit {
		id := registerMenuDo(w, func(win *Window) { win.Close() })
		title := C.CString(orTitle(it.Title, roleTitle(it.Role)))
		C.appWinAddRoleQuit(C.int(id), title)
		C.free(unsafe.Pointer(title))
		return
	}
	if it.Role != RoleNone {
		// Cut/Copy/Paste/Select All: accelerators still reach the webview
		// via the OS; the menu item posts the matching WM_* to the focus.
		id := registerMenuDo(w, func(*Window) { winRole(it.Role) })
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
		C.appWinAddMenuItem(C.int(id), title, key, shift, alt, ctrl)
		C.free(unsafe.Pointer(title))
		return
	}
	if it.Do == nil {
		return
	}
	id := registerMenuDo(w, it.Do)
	title := C.CString(it.Title)
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
	C.appWinAddMenuItem(C.int(id), title, key, shift, alt, ctrl)
	C.free(unsafe.Pointer(title))
}
