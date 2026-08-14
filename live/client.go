package live

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strconv"
	"sync"

	alacris "github.com/bmartel/alacris-go"
)

//go:embed assets/live.js
var clientFS embed.FS

// Client returns the live client script, for projects that would rather serve
// it from their own asset pipeline than from Go.
func Client() []byte {
	body, err := fs.ReadFile(clientFS, "assets/live.js")
	if err != nil {
		panic(err) // embedded at compile time
	}
	return body
}

type clientAsset struct {
	body   []byte
	etag   string
	gzBody []byte
	gzETag string
}

var clientOnce = sync.OnceValue(func() clientAsset {
	body := Client()
	sum := sha256.Sum256(body)
	tag := hex.EncodeToString(sum[:16])
	a := clientAsset{body: body, etag: `"` + tag + `"`}

	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := gz.Write(body); err == nil && gz.Close() == nil && buf.Len() < len(body) {
		a.gzBody = buf.Bytes()
		a.gzETag = `"` + tag + `-gz"`
	}
	return a
})

func serveClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	asset := clientOnce()

	h := w.Header()
	h.Set("Content-Type", "text/javascript; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")

	body, etag := asset.body, asset.etag
	if asset.gzBody != nil {
		h.Set("Vary", "Accept-Encoding")
		if alacris.AcceptsGzip(r) {
			body, etag = asset.gzBody, asset.gzETag
			h.Set("Content-Encoding", "gzip")
		}
	}

	alacris.SetCacheHeaders(h, r, etag)
	if alacris.MatchETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body)
}
