package live

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/a-h/templ"
)

// templText is a templ.Component that writes s verbatim.
func templText(s string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

// newSession stands in for a page render: NewSession needs an exchange,
// because that is where the cookie is set.
func newSession(srv *Server) *Session {
	rec := httptest.NewRecorder()
	return srv.NewSession(rec, httptest.NewRequest(http.MethodGet, "/", nil))
}

// newSessionSharing renders a second page in the same browser, by replaying the
// cookie the first render set. Two tabs, one cookie, two page ids.
func newSessionSharing(srv *Server, existing *Session) *Session {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(sessionCookie(existing))
	return srv.NewSession(rec, req)
}

// sessionCookie is the cookie a browser holding this session would send.
func sessionCookie(s *Session) *http.Cookie {
	return &http.Cookie{Name: s.srv.opts.CookieName, Value: s.client}
}

// tlsState marks a request as having arrived over TLS, which is all
// SecureAuto looks at.
var tlsState = tls.ConnectionState{}
