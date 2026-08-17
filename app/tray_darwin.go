//go:build desktop && darwin

package app

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
void appTrayInstall(const char *title, const char *tooltip);
void appTrayAddItem(int id, const char *title);
void appTrayAddSeparator(void);
void appTrayFinish(void);
*/
import "C"
import "unsafe"

func applyTray(w *Window, t *Tray) {
	if t == nil {
		return
	}
	title := t.Title
	if title == "" {
		title = "•"
	}
	ct := C.CString(title)
	tip := C.CString(t.Tooltip)
	C.appTrayInstall(ct, tip)
	C.free(unsafe.Pointer(ct))
	C.free(unsafe.Pointer(tip))
	if t.Menu != nil {
		for _, it := range t.Menu.Items {
			addTrayItem(w, it)
		}
	}
	C.appTrayFinish()
}

func addTrayItem(w *Window, it MenuItem) {
	if it.Role == RoleSeparator {
		C.appTrayAddSeparator()
		return
	}
	if it.Role == RoleQuit {
		id := registerMenuDo(w, func(win *Window) { win.Close() })
		title := C.CString(orTitle(it.Title, "Quit"))
		C.appTrayAddItem(C.int(id), title)
		C.free(unsafe.Pointer(title))
		return
	}
	if it.Do == nil {
		return
	}
	id := registerMenuDo(w, it.Do)
	title := C.CString(it.Title)
	C.appTrayAddItem(C.int(id), title)
	C.free(unsafe.Pointer(title))
}
