//go:build desktop && darwin

package app

/*
#cgo LDFLAGS: -framework Carbon -framework Cocoa
int appHotkeyRegister(int id, int key, int shift, int alt, int cmd);
void appHotkeyUnregister(int id);
*/
import "C"

func registerShortcut(w *Window, keys string, fn func()) error {
	k := parseKeys(keys)
	if k.key == "" {
		return ErrNoShortcut
	}
	id := nextHotID()
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
	if C.appHotkeyRegister(C.int(id), C.int(k.key[0]), shift, alt, cmd) == 0 {
		return ErrNoShortcut
	}
	storeHot(id, keys, fn)
	return nil
}

func unregisterShortcut(w *Window, keys string) error {
	id, ok := popHot(keys)
	if !ok {
		return nil
	}
	C.appHotkeyUnregister(C.int(id))
	return nil
}
