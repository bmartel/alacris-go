package live

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func quiet() Options {
	return Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// The capability is the cookie, so its attributes are the security boundary.
func TestSessionCookieAttributes(t *testing.T) {
	srv := New(quiet())
	defer srv.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
	req.TLS = &tlsState
	srv.NewSession(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	c := cookies[0]

	if c.Name != DefaultCookieName {
		t.Errorf("name %q", c.Name)
	}
	if !c.HttpOnly {
		t.Error("the cookie must be HttpOnly, or script can read the capability")
	}
	if !c.Secure {
		t.Error("the cookie must be Secure over TLS, or it travels in clear on a downgrade")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax: it is what stops a cross-site POST carrying it", c.SameSite)
	}
	if c.Path != "/_alacris/" {
		t.Errorf("path %q: the cookie should not be sent to the rest of the application", c.Path)
	}
	if !c.Expires.IsZero() || c.MaxAge != 0 {
		t.Error("the cookie should last the browser session, not longer")
	}
	if !validToken(c.Value) {
		t.Errorf("value %q is not a token", c.Value)
	}
	if vary := rec.Result().Header.Get("Vary"); !strings.Contains(vary, "Cookie") {
		t.Errorf("Vary = %q, want it to mention Cookie", vary)
	}
}

func TestCookieSecureModes(t *testing.T) {
	plain := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	tls := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
	tls.TLS = &tlsState
	proxied := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	proxied.Header.Set("X-Forwarded-Proto", "https")

	cases := []struct {
		name string
		mode Secure
		req  *http.Request
		want bool
	}{
		{"auto over plain http", SecureAuto, plain, false},
		{"auto over tls", SecureAuto, tls, true},
		{"auto behind a terminating proxy", SecureAuto, proxied, true},
		{"always", SecureAlways, plain, true},
		{"never", SecureNever, tls, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := quiet()
			o.CookieSecure = c.mode
			srv := New(o)
			defer srv.Close()

			rec := httptest.NewRecorder()
			srv.NewSession(rec, c.req)
			if got := rec.Result().Cookies()[0].Secure; got != c.want {
				t.Errorf("Secure = %v, want %v", got, c.want)
			}
		})
	}
}

// Mount knows where the endpoints are, so the cookie is scoped to them.
func TestMountScopesTheCookie(t *testing.T) {
	srv := New(quiet())
	defer srv.Close()

	Mount(http.NewServeMux(), "/app/_live/", srv)

	rec := httptest.NewRecorder()
	srv.NewSession(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rec.Result().Cookies()[0].Path; got != "/app/_live/" {
		t.Errorf("path %q, want the mount point", got)
	}
}

// One browser, two tabs. A cookie is per-browser and a session is per page
// render, so the page id is what keeps them apart — a cookie carrying the
// session id would have the second render overwrite the first.
func TestOneBrowserManyPages(t *testing.T) {
	srv, ts := newTestServer(t)

	first := newSession(srv)
	second := newSessionSharing(srv, first)

	if first.ID() == second.ID() {
		t.Fatal("two page renders produced one page id")
	}
	if first.client != second.client {
		t.Fatal("two pages in one browser should share a client token")
	}

	// Both are independently addressable, and neither disturbed the other.
	a, stopA := stream(t, ts, first)
	defer stopA()
	b, stopB := stream(t, ts, second)
	defer stopB()
	waitConnected(t, first)
	waitConnected(t, second)

	first.Element("x").Set("n", 1)
	if got := recv(t, a); len(got) != 1 || got[0].Value != float64(1) {
		t.Errorf("first page got %+v", got)
	}

	second.Element("x").Set("n", 2)
	if got := recv(t, b); len(got) != 1 || got[0].Value != float64(2) {
		t.Errorf("second page got %+v", got)
	}
}

// A page id is in the stream URL and therefore in access logs. On its own it
// has to be worth nothing.
func TestAPageIDWithoutTheCookieReachesNothing(t *testing.T) {
	srv, ts := newTestServer(t)
	sess := newSession(srv)
	srv.On("act", func(c *Ctx) error {
		t.Error("an unauthorised action ran")
		return nil
	})

	t.Run("stream", func(t *testing.T) {
		resp, err := ts.Client().Get(ts.URL + "/_alacris/live?p=" + sess.ID())
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status %d, want 404 without a cookie", resp.StatusCode)
		}
	})

	t.Run("action", func(t *testing.T) {
		post(t, ts, nil, `{"p":"`+sess.ID()+`","a":"act","d":null}`, http.StatusNotFound)
	})
}

// A cookie from one browser must not reach another browser's page, or the
// split buys nothing.
func TestACookieCannotDriveAnotherBrowsersPage(t *testing.T) {
	srv, ts := newTestServer(t)

	mine := newSession(srv)
	theirs := newSession(srv) // a different browser: its own token

	if mine.client == theirs.client {
		t.Fatal("two browsers were given the same token")
	}

	srv.On("act", func(c *Ctx) error {
		t.Error("an action ran for a page the cookie does not own")
		return nil
	})

	// My cookie, their page id.
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/_alacris/live?p="+theirs.ID(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(sessionCookie(mine))
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("stream status %d, want 404", resp.StatusCode)
	}

	post(t, ts, mine, `{"p":"`+theirs.ID()+`","a":"act","d":null}`, http.StatusNotFound)
}

// With the capability moved to an ambient credential, these are what keep a
// hostile page from forging an action. SameSite does the real work in a
// browser; these are the parts this server controls.
func TestActionCSRFDefences(t *testing.T) {
	srv, ts := newTestServer(t)
	sess := newSession(srv)
	srv.On("act", func(c *Ctx) error { return nil })

	send := func(t *testing.T, contentType, body string, cookie bool, want int) {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/_alacris/live", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if cookie {
			req.AddCookie(sessionCookie(sess))
		}
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
		if resp.StatusCode != want {
			t.Errorf("status %d, want %d", resp.StatusCode, want)
		}
	}

	valid := `{"p":"` + sess.ID() + `","a":"act","d":null}`

	t.Run("the real thing works", func(t *testing.T) {
		send(t, "application/json", valid, true, http.StatusNoContent)
	})
	t.Run("a charset parameter is fine", func(t *testing.T) {
		send(t, "application/json; charset=utf-8", valid, true, http.StatusNoContent)
	})

	// A cross-site form can only send these three, and none of them is JSON.
	// Refusing anything else means a form cannot reach a handler even if the
	// browser did attach the cookie.
	for _, ct := range []string{
		"text/plain",
		"application/x-www-form-urlencoded",
		"multipart/form-data; boundary=x",
		"",
	} {
		t.Run("form content type "+ct+" is refused", func(t *testing.T) {
			send(t, ct, valid, true, http.StatusUnsupportedMediaType)
		})
	}

	// The page id is in markup a hostile page cannot read, so requiring it is
	// a double submit token that does not depend on SameSite holding.
	t.Run("a missing page id is refused", func(t *testing.T) {
		send(t, "application/json", `{"a":"act","d":null}`, true, http.StatusNotFound)
	})
	t.Run("a guessed page id is refused", func(t *testing.T) {
		send(t, "application/json", `{"p":"`+newToken()+`","a":"act","d":null}`, true, http.StatusNotFound)
	})
}

// Cross-origin is off unless an origin is named, because a credentialed
// endpoint that reflects whatever it is sent is an open door.
func TestCORS(t *testing.T) {
	t.Run("no CORS headers without AllowOrigin", func(t *testing.T) {
		_, ts := newTestServer(t)
		req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/_alacris/live", nil)
		req.Header.Set("Origin", "https://other.example")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("an untrusted origin was allowed: %q", got)
		}
	})

	t.Run("a named origin gets a credentialed preflight", func(t *testing.T) {
		o := quiet()
		o.AllowOrigin = func(origin string, _ *http.Request) bool {
			return origin == "https://app.example"
		}
		_, ts := newTestServer(t, o)

		req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/_alacris/live", nil)
		req.Header.Set("Origin", "https://app.example")
		req.Header.Set("Access-Control-Request-Method", "POST")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		h := resp.Header
		if got := h.Get("Access-Control-Allow-Origin"); got != "https://app.example" {
			t.Errorf("Allow-Origin = %q; it must echo the origin, never *", got)
		}
		if got := h.Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("Allow-Credentials = %q, without which the cookie is not sent", got)
		}
		if got := h.Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Content-Type") {
			t.Errorf("Allow-Headers = %q, without which the JSON content type is refused", got)
		}
		if !strings.Contains(h.Get("Vary"), "Origin") {
			t.Errorf("Vary = %q, want it to mention Origin", h.Get("Vary"))
		}
	})
}
