package app

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHostGateRejectsMissingToken(t *testing.T) {
	t.Parallel()
	h := hostGate(okHandler(), "secret-token-value-ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestHostGateRejectsWrongToken(t *testing.T) {
	t.Parallel()
	h := hostGate(okHandler(), "secret-token-value-ok")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?"+hostQuery+"=nope", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestHostGateQuerySetsCookieAndRedirects(t *testing.T) {
	t.Parallel()
	token := "secret-token-value-ok"
	h := hostGate(okHandler(), token)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/board?"+hostQuery+"="+token, nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc == "" {
		t.Fatal("missing Location")
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get(hostQuery) != "" {
		t.Fatalf("redirect still has token: %s", loc)
	}
	if rec.Header().Get("Set-Cookie") == "" {
		t.Fatal("missing Set-Cookie")
	}
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == hostCookieName && c.Value == token && c.HttpOnly {
			found = true
		}
	}
	if !found {
		t.Fatalf("cookies = %+v", cookies)
	}
}

func TestHostGateCookiePasses(t *testing.T) {
	t.Parallel()
	token := "secret-token-value-ok"
	h := hostGate(okHandler(), token)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(hostCookie(token))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestGatedListenRoundTrip(t *testing.T) {
	t.Parallel()
	h, err := listen(Options{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		}),
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	res, err := http.Get(h.URL() + "/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("ungated GET = %d, want 403", res.StatusCode)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	res, err = client.Get(h.BootstrapURL("/"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("bootstrap GET = %d", res.StatusCode)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q", body)
	}

	res, err = client.Get(h.URL() + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || string(body) != "ok" {
		t.Fatalf("cookie GET = %d %q", res.StatusCode, body)
	}
}

func TestSameHostToken(t *testing.T) {
	t.Parallel()
	if !sameHostToken("abc", "abc") {
		t.Fatal("equal tokens")
	}
	if sameHostToken("abc", "abd") {
		t.Fatal("unequal tokens")
	}
	if sameHostToken("abc", "ab") {
		t.Fatal("length mismatch")
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}
