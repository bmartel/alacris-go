//go:build desktop && darwin

package app

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void appInstallMenu(void);
void appAddMenuItem(int id, const char* title, const char* key, int shift, int alt, int cmd);
void appAddRoleItem(const char* title, const char* key, const char* selector);
void appAddSeparator(void);
void appBeginSubmenu(const char* title);
void appEndSubmenu(void);
void appFinishMenu(void);
*/
import "C"
import (
	"strings"
	"sync"
	"unsafe"
)

var (
	menuMu  sync.Mutex
	menuFns = map[int]func(){}
	menuSeq int
)

func applyMenu(w *Window, m *Menu) {
	if m == nil || len(m.Items) == 0 {
		return
	}
	menuMu.Lock()
	menuFns = map[int]func(){}
	menuSeq = 1
	menuMu.Unlock()

	C.appInstallMenu()
	for _, it := range m.Items {
		addMenuItem(w, it)
	}
	C.appFinishMenu()
}

func addMenuItem(w *Window, it MenuItem) {
	if it.Role == RoleSeparator {
		C.appAddSeparator()
		return
	}
	if len(it.Items) > 0 {
		title := C.CString(it.Title)
		C.appBeginSubmenu(title)
		C.free(unsafe.Pointer(title))
		for _, child := range it.Items {
			addMenuItem(w, child)
		}
		C.appEndSubmenu()
		return
	}
	k := parseKeys(it.Keys)
	key := C.CString(strings.ToLower(k.key))
	defer C.free(unsafe.Pointer(key))

	switch it.Role {
	case RoleQuit, RoleCut, RoleCopy, RolePaste, RoleSelectAll:
		title := C.CString(orTitle(it.Title, roleTitle(it.Role)))
		sel := C.CString(roleSelector(it.Role))
		C.appAddRoleItem(title, key, sel)
		C.free(unsafe.Pointer(title))
		C.free(unsafe.Pointer(sel))
		return
	}
	if it.Do == nil {
		return
	}
	menuMu.Lock()
	id := menuSeq
	menuSeq++
	do := it.Do
	menuFns[id] = func() { do(w) }
	menuMu.Unlock()
	title := C.CString(it.Title)
	shift, alt, cmd := C.int(0), C.int(0), C.int(0)
	if k.shift {
		shift = 1
	}
	if k.alt {
		alt = 1
	}
	if k.ctrlOrCmd {
		cmd = 1
	}
	C.appAddMenuItem(C.int(id), title, key, shift, alt, cmd)
	C.free(unsafe.Pointer(title))
}

func orTitle(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func roleTitle(r Role) string {
	switch r {
	case RoleQuit:
		return "Quit"
	case RoleCut:
		return "Cut"
	case RoleCopy:
		return "Copy"
	case RolePaste:
		return "Paste"
	case RoleSelectAll:
		return "Select All"
	}
	return ""
}

func roleSelector(r Role) string {
	switch r {
	case RoleQuit:
		return "terminate:"
	case RoleCut:
		return "cut:"
	case RoleCopy:
		return "copy:"
	case RolePaste:
		return "paste:"
	case RoleSelectAll:
		return "selectAll:"
	}
	return ""
}

//export goMenuInvoke
func goMenuInvoke(id C.int) {
	menuMu.Lock()
	fn := menuFns[int(id)]
	menuMu.Unlock()
	if fn != nil {
		fn()
	}
}
