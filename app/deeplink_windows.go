//go:build windows

package app

import (
	"fmt"
	"os"
)

func registerDeepLinkPlatform(scheme, identifier string) {
	if !validScheme(scheme) {
		return
	}
	exe := currentExecutable()
	base := `HKCU\Software\Classes\` + scheme
	runSilent("reg", "add", base, "/ve", "/d", "URL:"+scheme, "/f")
	runSilent("reg", "add", base, "/v", "URL Protocol", "/d", "", "/f")
	runSilent("reg", "add", base+`\shell\open\command`, "/ve", "/d",
		fmt.Sprintf(`"%s" "%%1"`, exe), "/f")
	_ = identifier
	_ = os.ErrExist
}
