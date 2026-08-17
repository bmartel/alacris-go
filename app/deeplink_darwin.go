//go:build desktop && darwin

package app

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
void appDeepLinkInstall(void);
*/
import "C"

func registerDeepLinkPlatform(scheme, identifier string) {
	if !validScheme(scheme) {
		return
	}
	C.appDeepLinkInstall()
	_ = identifier
}
