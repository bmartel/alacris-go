//go:build desktop && darwin

package app

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
void appWindowMinimize(void *win);
void appWindowMaximize(void *win);
void appWindowUnmaximize(void *win);
void appWindowFullscreen(void *win);
void appWindowUnfullscreen(void *win);
void appWindowShow(void *win);
void appWindowHide(void *win);
void appWindowFocus(void *win);
void appWindowSetAlwaysOnTop(void *win, int on);
void appWindowSetDecorations(void *win, int on);
void appWindowSetPosition(void *win, int x, int y);
void appWindowPosition(void *win, int *x, int *y);
void appWindowCenter(void *win);
void appWindowSetBadge(const char *s);
*/
import "C"
import (
	"unsafe"
)

func windowMinimize(p unsafe.Pointer)     { C.appWindowMinimize(p) }
func windowMaximize(p unsafe.Pointer)     { C.appWindowMaximize(p) }
func windowUnmaximize(p unsafe.Pointer)   { C.appWindowUnmaximize(p) }
func windowFullscreen(p unsafe.Pointer)   { C.appWindowFullscreen(p) }
func windowUnfullscreen(p unsafe.Pointer) { C.appWindowUnfullscreen(p) }
func windowShow(p unsafe.Pointer)         { C.appWindowShow(p) }
func windowHide(p unsafe.Pointer)         { C.appWindowHide(p) }
func windowFocus(p unsafe.Pointer)        { C.appWindowFocus(p) }
func windowSetAlwaysOnTop(p unsafe.Pointer, on bool) {
	C.appWindowSetAlwaysOnTop(p, boolInt(on))
}
func windowSetDecorations(p unsafe.Pointer, on bool) {
	C.appWindowSetDecorations(p, boolInt(on))
}
func windowSetPosition(p unsafe.Pointer, x, y int) {
	C.appWindowSetPosition(p, C.int(x), C.int(y))
}
func windowPosition(p unsafe.Pointer) (int, int) {
	var x, y C.int
	C.appWindowPosition(p, &x, &y)
	return int(x), int(y)
}
func windowCenter(p unsafe.Pointer) { C.appWindowCenter(p) }
func windowSetBadge(s string) {
	cs := C.CString(s)
	C.appWindowSetBadge(cs)
	C.free(unsafe.Pointer(cs))
}

func boolInt(on bool) C.int {
	if on {
		return 1
	}
	return 0
}
