//go:build desktop && !windows && !darwin

package app

/*
#cgo pkg-config: gtk+-3.0
#include <stdlib.h>
void appGtkTrayInstall(const char *title, const char *tooltip);
void appGtkTrayAddItem(int id, const char *title);
void appGtkTrayAddSeparator(void);
void appGtkTrayFinish(void);
*/
import "C"
import "unsafe"

func applyTray(w *Window, t *Tray) {
	if t == nil {
		return
	}
	title := t.Title
	if title == "" {
		title = "alacris"
	}
	ct := C.CString(title)
	tip := C.CString(t.Tooltip)
	C.appGtkTrayInstall(ct, tip)
	C.free(unsafe.Pointer(ct))
	C.free(unsafe.Pointer(tip))
	if t.Menu != nil {
		for _, it := range t.Menu.Items {
			addGtkTrayItem(w, it)
		}
	}
	C.appGtkTrayFinish()
}

func addGtkTrayItem(w *Window, it MenuItem) {
	if it.Role == RoleSeparator {
		C.appGtkTrayAddSeparator()
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
	C.appGtkTrayAddItem(C.int(id), title)
	C.free(unsafe.Pointer(title))
}
