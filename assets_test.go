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
	"alacris.js":     "4a01a83b08d6a1a59da73b3f0d02b5c8f3e1c4fb528720924fccf3c2a83ec5cd",
	"alacris.dev.js": "d222b06a7a6f8f5e4c6056fc57e6a9c490beddd93763ce012d509072f007f6ab",
	"store.js":       "e670d6bd36ad92069424718cd0f841322b6fc82283edab3ac6d1a9e883883024",
	"context.js":     "30ef84ce7edbaa30ff59a7d212f90a670da573cfd4e5d6d9ed5acf513be3d51d",
	"signal.js":      "3ceac06dbe4efed9ec9360d0a35d7e53428f7ea113fe01685b8d442bb8a73948",
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
