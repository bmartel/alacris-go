package live

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
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
	body []byte
	etag string
}

var clientOnce = sync.OnceValue(func() clientAsset {
	body := Client()
	sum := sha256.Sum256(body)
	return clientAsset{body: body, etag: `"` + hex.EncodeToString(sum[:16]) + `"`}
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
	alacris.SetCacheHeaders(h, r, asset.etag)
	if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, asset.etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Length", strconv.Itoa(len(asset.body)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(asset.body)
}
