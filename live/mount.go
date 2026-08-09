package live

import (
	"net/http"
	"strings"

	alacris "github.com/bmartel/alacris-go"
)

// Mount registers everything a live page needs on a ServeMux: the alacris
// runtime, the live client, and the patch and action endpoints.
//
//	mux := http.NewServeMux()
//	live.Mount(mux, alacris.DefaultBase, srv)
//
// base must match the Base in the alacris.Config the page renders with.
// Passing an empty base uses alacris.DefaultBase.
//
// The routes are ordinary patterns, so mounting them by hand is fine too —
// this only exists so that the three of them cannot drift apart.
func Mount(mux *http.ServeMux, base string, srv *Server) {
	if base == "" {
		base = alacris.DefaultBase
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	// The cookie is scoped to where the endpoints actually live, so it is not
	// attached to every other request the application serves.
	srv.setCookiePath(base)

	// The more specific patterns win over the prefix, so the runtime handler
	// serves everything except the two live endpoints.
	mux.Handle(base, alacris.RuntimeHandler())
	mux.Handle(base+"live", srv)
	mux.Handle(base+"live.js", srv)
}
