package live

import (
	"net/http"
	"net/url"
	"strings"
)

// Cross-origin support exists for one deployment: the page on one origin, the
// live endpoint on another. It is off unless Options.AllowOrigin says which
// origins are trusted, because a credentialed endpoint that reflects any
// origin is an open door.
//
// That deployment also needs Options.CookieSameSite set to
// http.SameSiteNoneMode, which browsers only honour with Secure. Without CORS
// headers the browser would refuse the response and the page would see a
// connection that never opens, which is a miserable thing to debug — so when
// an origin is allowed, the headers that make it work are sent.

// allowCORS echoes the request's origin when it is a trusted one.
func (s *Server) allowCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" || !s.crossOrigin(r, origin) {
		return
	}
	h := w.Header()
	// The specific origin, never "*": a wildcard is invalid with credentials,
	// and credentials are the whole point here.
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Access-Control-Allow-Credentials", "true")
	addVary(h, "Origin")
}

// crossOrigin reports whether origin is a different origin that is allowed.
func (s *Server) crossOrigin(r *http.Request, origin string) bool {
	if sameOrigin(origin, r) {
		return false
	}
	return s.opts.AllowOrigin != nil && s.opts.AllowOrigin(origin, r)
}

// servePreflight answers the OPTIONS a cross-origin action triggers.
//
// The action posts application/json, which is not a CORS-safelisted content
// type, so the browser asks first. That is deliberate: requiring a content
// type that cannot be set by a cross-site form is what keeps a form from
// forging an action.
func (s *Server) servePreflight(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" || !s.crossOrigin(r, origin) {
		// Nothing to preflight for a same-origin request, and nothing to offer
		// an origin that is not allowed.
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h := w.Header()
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Access-Control-Allow-Credentials", "true")
	h.Set("Access-Control-Allow-Methods", "GET, POST")
	h.Set("Access-Control-Allow-Headers", "Content-Type")
	h.Set("Access-Control-Max-Age", "600")
	addVary(h, "Origin")
	w.WriteHeader(http.StatusNoContent)
}

// sameOrigin reports whether origin names the host this request was sent to.
func sameOrigin(origin string, r *http.Request) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}
