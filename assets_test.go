package alacris

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// vendoredHashes pins the exact bytes of the alacris runtime published as
// npm alacris@RuntimeVersion.
//
// The point is not integrity — the module is what it is — but drift. A
// vendored copy of someone else's build silently goes stale; a failure here
// means `go run ./internal/vendorjs` needs running and RuntimeVersion needs
// bumping, which is exactly the moment to check the changelog.
var vendoredHashes = map[string]string{
	"alacris.js":     "624569d4f6888e7f3f77222c3f7abde70a45ff733a82ca2d4b477cc92cc2bab1",
	"alacris.dev.js": "336bfcd350e788b44b9d10c55be3b193972a4cebeda093c0fec855ab35651764",
	"store.js":       "1de9821cd7d9663aa8f1b139f2373a16df73f2cb248192faf8c3fb9d66ab2e99",
	"context.js":     "4515dfdff5bb66e2430799c55d9327e7510bf94ba594de033c7d3522731d6411",
	"signal.js":      "6f8d7672dbd9b754f06b9b7c77bb1f97021374d2173a94205246b04904260e52",
}

func TestVendoredAssets(t *testing.T) {
	assets := Assets()
	for name, want := range vendoredHashes {
		body, err := fs.ReadFile(assets, name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		sum := sha256.Sum256(body)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("%s: hash %s, want %s\n\trun: go run ./internal/vendorjs", name, got, want)
		}
	}

	if _, err := fs.ReadFile(assets, "LICENSE.alacris"); err != nil {
		t.Errorf("the vendored runtime must ship its license: %v", err)
	}
}

func TestRuntimeHandler(t *testing.T) {
	h := RuntimeHandler()

	t.Run("serves by file name at any mount point", func(t *testing.T) {
		for _, path := range []string{"/_alacris/alacris.js", "/static/js/alacris.js", "/alacris.js"} {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s: status %d", path, rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
				t.Errorf("GET %s: content type %q", path, ct)
			}
			if !strings.Contains(rec.Body.String(), "customElements") {
				t.Errorf("GET %s: body does not look like the runtime", path)
			}
		}
	})

	t.Run("revalidates with an etag", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_alacris/store.js", nil))
		etag := rec.Header().Get("ETag")
		if etag == "" {
			t.Fatal("no ETag")
		}

		req := httptest.NewRequest(http.MethodGet, "/_alacris/store.js", nil)
		req.Header.Set("If-None-Match", etag)
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotModified {
			t.Errorf("status %d, want 304", rec.Code)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("304 carried a body")
		}
	})

	t.Run("refuses anything not vendored", func(t *testing.T) {
		for _, path := range []string{"/_alacris/../../etc/passwd", "/_alacris/nope.js", "/_alacris/index.html"} {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusNotFound {
				t.Errorf("GET %s: status %d, want 404", path, rec.Code)
			}
		}
	})

	t.Run("rejects writes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/_alacris/alacris.js", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("status %d, want 405", rec.Code)
		}
	})
}
