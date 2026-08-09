package alacris

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/a-h/templ"
)

//go:embed assets
var assetFS embed.FS

// RuntimeVersion is the version of the alacris npm package vendored in
// assets/. Regenerate with `go generate ./...` after bumping it.
const RuntimeVersion = "0.2.2"

// TrustedTypesPolicy is the Trusted Types policy name alacris registers for
// template parsing. Under a trusted-types CSP directive it has to be allowed.
const TrustedTypesPolicy = "alacris"

// Asset file names served by RuntimeHandler.
const (
	AssetCore    = "alacris.js"
	AssetCoreDev = "alacris.dev.js"
	AssetStore   = "store.js"
	AssetContext = "context.js"
	AssetSignal  = "signal.js"
	AssetLive    = "live.js"
)

// DefaultBase is where the runtime is expected to be mounted.
const DefaultBase = "/_alacris/"

// Assets returns the vendored alacris runtime as a filesystem, for projects
// that would rather copy the files into their own asset pipeline than serve
// them from Go.
func Assets() fs.FS {
	sub, err := fs.Sub(assetFS, "assets")
	if err != nil {
		panic(err) // the embedded tree is fixed at compile time
	}
	return sub
}

// RuntimeHandler serves the vendored alacris runtime.
//
// It resolves requests by file name only, so it behaves the same at any mount
// point and does not need http.StripPrefix:
//
//	mux.Handle("/_alacris/", alacris.RuntimeHandler())
func RuntimeHandler() http.Handler {
	assets := map[string]*asset{}

	// Read and hash everything once, here. The set of files is fixed at
	// compile time, so caching on demand bought nothing and cost something:
	// a miss was remembered too, which let a few thousand requests for
	// made-up names grow a map that nothing ever emptied. A map built once
	// and only read afterwards also needs no lock on the serving path.
	entries, err := fs.ReadDir(Assets(), ".")
	if err != nil {
		panic(err) // the embedded tree is fixed at compile time
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".js") {
			continue
		}
		body, err := fs.ReadFile(Assets(), name)
		if err != nil {
			panic(err)
		}
		sum := sha256.Sum256(body)
		assets[name] = &asset{body: body, etag: `"` + hex.EncodeToString(sum[:16]) + `"`}
	}

	return &runtimeHandler{assets: assets}
}

type runtimeHandler struct {
	// assets is written once by RuntimeHandler and only read afterwards, so
	// it needs no synchronisation.
	assets map[string]*asset
}

type asset struct {
	body []byte
	etag string
}

func (h *runtimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a, ok := h.assets[path.Base(path.Clean("/"+r.URL.Path))]
	if !ok {
		http.NotFound(w, r)
		return
	}
	head := w.Header()
	head.Set("Content-Type", "text/javascript; charset=utf-8")
	SetCacheHeaders(head, r, a.etag)
	if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, a.etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	head.Set("Content-Length", itoa(len(a.body)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(a.body)
}

// SetCacheHeaders applies the caching policy for a served asset.
//
// An asset URL with no version in it must be revalidated: it is the same URL
// before and after a deploy, so a long max-age means a fixed bug stays broken
// in every browser that already has the old copy. Revalidation costs one
// conditional request and almost always answers 304.
//
// Config.Version puts a ?v= on the URLs it emits, and that is what makes an
// asset safe to cache for a year: a new version is a new URL.
func SetCacheHeaders(h http.Header, r *http.Request, etag string) {
	h.Set("ETag", etag)
	if r.URL.Query().Get("v") != "" {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	h.Set("Cache-Control", "public, no-cache")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// Config describes the script tags a page needs.
type Config struct {
	// Base is the URL prefix RuntimeHandler is mounted at. Defaults to
	// DefaultBase.
	Base string

	// Dev swaps in the unminified build, which carries readable errors.
	Dev bool

	// Modules are your own ES module entry points, loaded after the import
	// map. This is where the file that calls define() for your components
	// goes.
	Modules []string

	// Imports adds or overrides import map entries, for pulling in component
	// packages of your own by bare specifier.
	Imports map[string]string

	// Live loads the live client and connects it. Page must be the id of a
	// session created by the live package for this page render.
	//
	// The page id is not a secret and does not need protecting: it says which
	// of a browser's pages is talking, and it is useless without the cookie
	// the live package sets, which script cannot read and which never appears
	// in a page or a URL.
	Live bool
	Page string

	// Endpoint is where the live client connects. Defaults to Base + "live".
	Endpoint string

	// Nonce is the CSP nonce for the emitted script tags. When empty, the
	// nonce carried on the context by templ.WithNonce is used.
	Nonce string

	// Version is appended to every asset URL as ?v=, which makes each release
	// a distinct URL and lets the handler cache it for a year instead of
	// revalidating it. Any string that changes with a deploy works: a release
	// tag, a build id, a commit.
	Version string
}

// asset returns the URL for one served file.
func (c Config) asset(name string) string {
	u := c.base() + name
	if c.Version != "" {
		u += "?v=" + url.QueryEscape(c.Version)
	}
	return u
}

func (c Config) base() string {
	b := c.Base
	if b == "" {
		b = DefaultBase
	}
	if !strings.HasSuffix(b, "/") {
		b += "/"
	}
	return b
}

func (c Config) endpoint() string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	return strings.TrimSuffix(c.base(), "/") + "/live"
}

// ImportMap returns the import map this configuration produces, so the same
// specifiers can be reused by a bundler or a test.
func (c Config) ImportMap() map[string]string {
	core := AssetCore
	if c.Dev {
		core = AssetCoreDev
	}
	imports := map[string]string{
		"alacris":         c.asset(core),
		"alacris/store":   c.asset(AssetStore),
		"alacris/context": c.asset(AssetContext),
		"alacris/signal":  c.asset(AssetSignal),
	}
	for k, v := range c.Imports {
		imports[k] = v
	}
	return imports
}

// Scripts renders the import map, your module entry points, and — when
// configured — the live client.
//
// It belongs in <head>, before any other module script: an import map has to
// precede the first module import it applies to.
func (c Config) Scripts() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		nonce := c.Nonce
		if nonce == "" {
			nonce = templ.GetNonce(ctx)
		}
		nonceAttr := ""
		if nonce != "" {
			nonceAttr = ` nonce="` + templ.EscapeString(nonce) + `"`
		}

		// json.Marshal escapes <, > and & to \u00xx, so the map cannot close
		// the script element early.
		body, err := json.Marshal(struct {
			Imports map[string]string `json:"imports"`
		}{Imports: c.ImportMap()})
		if err != nil {
			return err
		}

		var b strings.Builder
		b.WriteString(`<script type="importmap"` + nonceAttr + `>`)
		b.Write(body)
		b.WriteString("</script>")

		for _, m := range c.Modules {
			b.WriteString(`<script type="module" src="` + templ.EscapeString(m) + `"` + nonceAttr + `></script>`)
		}

		if c.Live {
			b.WriteString(`<script type="module" src="` + templ.EscapeString(c.asset(AssetLive)) + `"` +
				` data-endpoint="` + templ.EscapeString(c.endpoint()) + `"` +
				` data-page="` + templ.EscapeString(c.Page) + `"` + nonceAttr + `></script>`)
		}

		_, err = io.WriteString(w, b.String())
		return err
	})
}

// Scripts is shorthand for cfg.Scripts().
func Scripts(cfg Config) templ.Component { return cfg.Scripts() }
