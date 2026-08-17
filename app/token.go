package app

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

// hostCookieName is the capability that closes the loopback port. EventSource
// requires an HTTP family URL, so the webview cannot use a custom scheme; the
// token is issued only to that webview (as a query on the first navigation,
// then an HttpOnly cookie). Another local process that can reach the port
// still cannot create a session without it.
const (
	hostCookieName = "alacris_host"
	hostQuery      = "alacris_host"
	hostTokenBytes = 18
)

func newHostToken() string {
	var b [hostTokenBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("app: no randomness available: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func sameHostToken(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func hostCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     hostCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

// hostGate requires the host token as a cookie, or as a one-shot query
// parameter that is swapped for that cookie and stripped from the URL.
func hostGate(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(hostCookieName); err == nil && sameHostToken(c.Value, token) {
			next.ServeHTTP(w, r)
			return
		}
		q := r.URL.Query().Get(hostQuery)
		if q != "" && sameHostToken(q, token) {
			http.SetCookie(w, hostCookie(token))
			u := *r.URL
			vals := u.Query()
			vals.Del(hostQuery)
			u.RawQuery = vals.Encode()
			http.Redirect(w, r, u.RequestURI(), http.StatusFound)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	})
}
