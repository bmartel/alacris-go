//go:build desktop

package app

import "C"
import "unsafe"

var deepLinkFn func(string)

func setDeepLinkHandler(fn func(string)) {
	deepLinkFn = fn
}

//export goDeepLink
func goDeepLink(url *C.char) {
	if deepLinkFn != nil && url != nil {
		deepLinkFn(C.GoString(url))
	}
	_ = unsafe.Sizeof(url)
}
