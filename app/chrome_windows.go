//go:build desktop && windows

package app

/*
#cgo LDFLAGS: -luser32 -lshell32 -lgdi32
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
void appWindowSetTitlebar(void *win, int style);
void appWindowBeginDrag(void *win);
void appWindowSetPosition(void *win, int x, int y);
void appWindowPosition(void *win, int *x, int *y);
void appWindowCenter(void *win);
*/
import "C"
import "unsafe"

func windowMinimize(p unsafe.Pointer)   { C.appWindowMinimize(p) }
func windowMaximize(p unsafe.Pointer)   { C.appWindowMaximize(p) }
func windowUnmaximize(p unsafe.Pointer) { C.appWindowUnmaximize(p) }
func windowFullscreen(p unsafe.Pointer) { C.appWindowFullscreen(p) }
func windowUnfullscreen(p unsafe.Pointer) {
	C.appWindowUnfullscreen(p)
}
func windowShow(p unsafe.Pointer)  { C.appWindowShow(p) }
func windowHide(p unsafe.Pointer)  { C.appWindowHide(p) }
func windowFocus(p unsafe.Pointer) { C.appWindowFocus(p) }
func windowSetAlwaysOnTop(p unsafe.Pointer, on bool) {
	v := C.int(0)
	if on {
		v = 1
	}
	C.appWindowSetAlwaysOnTop(p, v)
}
func windowSetDecorations(p unsafe.Pointer, on bool) {
	v := C.int(0)
	if on {
		v = 1
	}
	C.appWindowSetDecorations(p, v)
}
func windowSetTitlebar(p unsafe.Pointer, style int) {
	C.appWindowSetTitlebar(p, C.int(style))
}
func windowBeginDrag(p unsafe.Pointer) { C.appWindowBeginDrag(p) }

func windowSetPosition(p unsafe.Pointer, x, y int) {
	C.appWindowSetPosition(p, C.int(x), C.int(y))
}
func windowPosition(p unsafe.Pointer) (int, int) {
	var x, y C.int
	C.appWindowPosition(p, &x, &y)
	return int(x), int(y)
}
func windowCenter(p unsafe.Pointer) { C.appWindowCenter(p) }
func windowSetBadge(string)         {}
