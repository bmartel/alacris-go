//go:build desktop && windows

package app

/*
#cgo LDFLAGS: -luser32
int appWinHotkeyRegister(void *hwnd, int id, int vk, int mods);
void appWinHotkeyUnregister(void *hwnd, int id);
*/
import "C"
import (
	"strings"
	"unsafe"
)

func registerShortcut(w *Window, keys string, fn func()) error {
	k := parseKeys(keys)
	if k.key == "" || w == nil {
		return ErrNoShortcut
	}
	hwnd := unsafe.Pointer(w.handle())
	if hwnd == nil {
		return ErrNoShortcut
	}
	id := nextHotID()
	mods := C.int(0)
	if k.ctrlOrCmd {
		mods |= C.int(0x0002) // MOD_CONTROL
	}
	if k.shift {
		mods |= C.int(0x0004) // MOD_SHIFT
	}
	if k.alt {
		mods |= C.int(0x0001) // MOD_ALT
	}
	vk := C.int(strings.ToUpper(k.key)[0])
	if C.appWinHotkeyRegister(hwnd, C.int(id), vk, mods) == 0 {
		return ErrNoShortcut
	}
	storeHot(id, keys, fn)
	return nil
}

func unregisterShortcut(w *Window, keys string) error {
	id, ok := popHot(keys)
	if !ok || w == nil {
		return nil
	}
	hwnd := unsafe.Pointer(w.handle())
	C.appWinHotkeyUnregister(hwnd, C.int(id))
	return nil
}
