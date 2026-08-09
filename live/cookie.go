package live

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

// A live session is identified by two values, and both are required.
//
//   - The client token lives in a cookie. It is the capability: HttpOnly so
//     script cannot read it, SameSite so it is not sent cross-site, and scoped
//     by path so it is not sent to the rest of the application. It never
//     appears in a URL, a page, or a log.
//
//   - The page id lives in the page and travels in the stream's query string.
//     It says which of a browser's open pages is talking. On its own it grants
//     nothing.
//
// The split exists because a cookie is per-browser and a session is per page
// render. Two tabs share one cookie, so a cookie carrying the session id would
// have the second tab's render overwrite the first tab's session. One cookie
// and many page ids keeps tabs independent.
//
// It also means neither value alone is enough. The page id is in a URL and
// therefore in access logs; the cookie is not, and cannot be read by script.
// An attacker needs both.

// DefaultCookieName is the cookie the client token is stored in.
const DefaultCookieName = "alacris_live"

// tokenBytes is the length of both the client token and a page id. 144 bits of
// randomness each, so neither is worth guessing.
const tokenBytes = 18

// Secure decides whether the session cookie carries the Secure attribute.
type Secure int

const (
	// SecureAuto sets Secure when the request arrived over TLS, directly or
	// through a proxy that said so. It is the default: correct in production,
	// and it does not break plain-HTTP development.
	SecureAuto Secure = iota

	// SecureAlways sets it unconditionally. Use it behind a proxy that
	// terminates TLS without setting X-Forwarded-Proto.
	SecureAlways

	// SecureNever omits it. Only sensible for local development, and never
	// with SameSite=None, which browsers reject without Secure.
	SecureNever
)

func newToken() string {
	var b [tokenBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail on any supported platform, and carrying on
		// with a guessable token would hand one browser's pages to whoever
		// guesses it.
		panic("live: no randomness available: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

// validToken rejects a cookie value that is not shaped like one we issued.
// It does not prove we issued it — nothing does, and nothing needs to: a
// session is bound to whatever token was present when it was created.
func validToken(s string) bool {
	if len(s) != base64.RawURLEncoding.EncodedLen(tokenBytes) {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(s)
	return err == nil
}

// sameToken compares in constant time. The comparison decides whether a
// request may drive a page, so it should not leak the answer through timing.
func sameToken(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// clientToken returns the token the request carries, if it carries a usable one.
func (s *Server) clientToken(r *http.Request) (string, bool) {
	c, err := r.Cookie(s.opts.CookieName)
	if err != nil || !validToken(c.Value) {
		return "", false
	}
	return c.Value, true
}

// issueToken returns the request's client token, minting one and setting the
// cookie when the browser does not have a usable one yet.
func (s *Server) issueToken(w http.ResponseWriter, r *http.Request) string {
	if token, ok := s.clientToken(r); ok {
		return token
	}

	token := newToken()
	http.SetCookie(w, &http.Cookie{
		Name:     s.opts.CookieName,
		Value:    token,
		Path:     s.opts.CookiePath,
		Domain:   s.opts.CookieDomain,
		HttpOnly: true,
		Secure:   s.secureCookie(r),
		SameSite: s.opts.CookieSameSite,
		// No Expires or MaxAge: it lasts as long as the browser session, which
		// is longer than any page that uses it and shorter than a machine.
	})

	// A page render that sets a cookie must not be cached by anything in
	// between, or the next visitor is handed this browser's token.
	addVary(w.Header(), "Cookie")
	return token
}

func (s *Server) secureCookie(r *http.Request) bool {
	switch s.opts.CookieSecure {
	case SecureAlways:
		return true
	case SecureNever:
		return false
	default:
		return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	}
}

// addVary appends to Vary without duplicating what is already there.
func addVary(h http.Header, value string) {
	for _, existing := range h.Values("Vary") {
		for _, part := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	h.Add("Vary", value)
}
