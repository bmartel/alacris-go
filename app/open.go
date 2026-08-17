package app

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// deniedOpenSchemes name handlers that reach a local file, a network share, or
// a script engine. OpenURL forwards to the OS opener, and a web page that can
// influence the argument must not be able to make it open file:// or smb://
// locations or invoke a script scheme.
var deniedOpenSchemes = map[string]bool{
	"file": true, "smb": true, "nfs": true, "afp": true, "dav": true, "davs": true,
	"javascript": true, "vbscript": true, "data": true, "chrome": true, "about": true,
}

// OpenURL opens raw in the user's default handler. http, https, mailto, and
// registered app schemes are accepted; file, network-share and script schemes
// are refused, as is anything that is not a syntactically valid URL.
func OpenURL(ctx context.Context, raw string) error {
	scheme, ok := urlScheme(raw)
	if !ok {
		return fmt.Errorf("app: OpenURL: %q is not a valid URL", raw)
	}
	if deniedOpenSchemes[scheme] {
		return fmt.Errorf("app: OpenURL: the %q scheme is not allowed", scheme)
	}
	return openURL(ctx, raw)
}

// urlScheme returns the lower-cased scheme of raw when raw is a valid absolute
// URL: an RFC-3986 scheme (letter, then letters/digits/+-.) followed by "://".
// Requiring valid scheme syntax also rejects a leading '-', which would
// otherwise reach the OS opener as an option rather than an operand.
func urlScheme(raw string) (string, bool) {
	i := strings.Index(raw, "://")
	if i <= 0 {
		return "", false
	}
	scheme := raw[:i]
	for j, r := range scheme {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case j > 0 && (r >= '0' && r <= '9' || r == '+' || r == '-' || r == '.'):
		default:
			return "", false
		}
	}
	return strings.ToLower(scheme), true
}

// RevealInFileManager shows path in the OS file manager.
func RevealInFileManager(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("app: RevealInFileManager: empty path")
	}
	return reveal(ctx, path)
}

var openExec = func(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}
