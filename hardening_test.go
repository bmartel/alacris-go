package alacris

import (
	"compress/gzip"
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// An escaped value is no defence for an event handler attribute: the browser
// executes it as script either way. So the name class is refused outright, on
// both the attr path and the prop path.
func TestEventHandlerAttributesAreRejected(t *testing.T) {
	for _, name := range []string{"onclick", "onClick", "ONERROR", "onpointerdown"} {
		if err := E("x-y").Attr(name, "alert(1)").Err(); err == nil {
			t.Errorf("Attr(%q) was accepted; a handler attribute executes its value", name)
		}
	}

	// On the prop path only an already-lowercase name survives as a handler
	// attribute — kebab-casing turns "onClick" into the inert "on-click".
	if err := E("x-y").Prop("onclick", "alert(1)").Err(); err == nil {
		t.Error(`Prop("onclick") was accepted; it renders a handler attribute verbatim`)
	}
	if err := E("x-y").Prop("onClick", "v").Err(); err != nil {
		t.Errorf(`Prop("onClick") was rejected, but its attribute "on-click" is inert: %v`, err)
	}

	// Names that merely start with "on" are not handlers and must keep working.
	for _, name := range []string{"on-line", "one-two", "on2x"} {
		if err := E("x-y").Attr(name, "v").Err(); err != nil {
			t.Errorf("Attr(%q) was rejected: %v", name, err)
		}
	}
}

func TestMatchETag(t *testing.T) {
	cases := []struct {
		header, etag string
		want         bool
	}{
		{`"abc"`, `"abc"`, true},
		{`W/"abc"`, `"abc"`, true},
		{`"zzz", "abc"`, `"abc"`, true},
		{`*`, `"abc"`, true},
		// The substring bug MatchETag replaced: a header that merely contains
		// the etag's text is not a match.
		{`"abcd"`, `"abc"`, false},
		{`"zzz"`, `"abc"`, false},
		{``, `"abc"`, false},
		{`"abc"`, ``, false},
	}
	for _, c := range cases {
		if got := MatchETag(c.header, c.etag); got != c.want {
			t.Errorf("MatchETag(%q, %q) = %v, want %v", c.header, c.etag, got, c.want)
		}
	}
}

func TestAcceptsGzip(t *testing.T) {
	cases := []struct {
		header string
		want   bool
	}{
		{"gzip", true},
		{"gzip, deflate, br", true},
		{"br;q=1.0, gzip;q=0.8", true},
		{"*", true},
		{"gzip;q=0", false},
		{"identity", false},
		{"", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if c.header != "" {
			r.Header.Set("Accept-Encoding", c.header)
		}
		if got := AcceptsGzip(r); got != c.want {
			t.Errorf("AcceptsGzip(%q) = %v, want %v", c.header, got, c.want)
		}
	}
}

func TestRuntimeAssetsAreServedHardenedAndCompressed(t *testing.T) {
	h := RuntimeHandler()

	get := func(mod func(*http.Request)) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/_alacris/alacris.js", nil)
		if mod != nil {
			mod(r)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}

	t.Run("nosniff on the plain response", func(t *testing.T) {
		rec := get(nil)
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
		}
		if got := rec.Header().Get("Content-Encoding"); got != "" {
			t.Errorf("a request that did not ask for gzip got Content-Encoding %q", got)
		}
	})

	t.Run("gzip is negotiated and round-trips", func(t *testing.T) {
		rec := get(func(r *http.Request) { r.Header.Set("Accept-Encoding", "gzip") })
		if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("Content-Encoding = %q, want gzip", got)
		}
		if got := rec.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
			t.Errorf("Vary = %q, want it to name Accept-Encoding", got)
		}
		gz, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(gz)
		if err != nil {
			t.Fatal(err)
		}
		want, err := fs.ReadFile(Assets(), AssetCore)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != string(want) {
			t.Error("the gzipped body does not decompress to the vendored file")
		}
	})

	t.Run("each representation revalidates against its own etag", func(t *testing.T) {
		plain := get(nil).Header().Get("ETag")
		zipped := get(func(r *http.Request) { r.Header.Set("Accept-Encoding", "gzip") }).Header().Get("ETag")
		if plain == "" || zipped == "" || plain == zipped {
			t.Fatalf("etags: plain %q, gzip %q; want two distinct strong validators", plain, zipped)
		}
		rec := get(func(r *http.Request) {
			r.Header.Set("Accept-Encoding", "gzip")
			r.Header.Set("If-None-Match", zipped)
		})
		if rec.Code != http.StatusNotModified {
			t.Errorf("revalidating the gzip representation answered %d, want 304", rec.Code)
		}
	})
}

// A pooled buffer must never leak one render's bytes into another.
func TestRenderIsStableAcrossPooledBuffers(t *testing.T) {
	want := `<x-y name="Ada"></x-y>`
	for i := 0; i < 100; i++ {
		var b strings.Builder
		if err := E("x-y").Prop("name", "Ada").Render(context.Background(), &b); err != nil {
			t.Fatal(err)
		}
		if b.String() != want {
			t.Fatalf("render %d = %q, want %q", i, b.String(), want)
		}
	}
}
