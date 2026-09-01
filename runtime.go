package alacris

import (
	"bytes"
	"compress/gzip"
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
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

//go:embed assets
var assetFS embed.FS

// RuntimeVersion is the version of the alacris npm package vendored in
// assets/. Regenerate with `go generate ./...` after bumping it.
const RuntimeVersion = "0.11.5"

// UIVersion is the version of the @alacris/ui package vendored in assets/ui/.
const UIVersion = "0.5.2"

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
	AssetUI      = "ui/index.js"
	AssetUITheme = "ui/theme/index.js"
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
	err := fs.WalkDir(Assets(), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(name, ".js") {
			return nil
		}
		body, err := fs.ReadFile(Assets(), name)
		if err != nil {
			return err
		}
		assets[name] = newAsset(body)
		return nil
	})
	if err != nil {
		panic(err) // the embedded tree is fixed at compile time
	}

	return &runtimeHandler{assets: assets}
}

// newAsset hashes and pre-compresses one served file. Compressing here, once,
// costs a few milliseconds at startup and saves re-compressing (or shipping
// identity-encoded script, 3-4x the bytes) on every request after it.
func newAsset(body []byte) *asset {
	sum := sha256.Sum256(body)
	tag := hex.EncodeToString(sum[:16])
	a := &asset{body: body, etag: `"` + tag + `"`}

	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := gz.Write(body); err == nil && gz.Close() == nil && buf.Len() < len(body) {
		a.gzBody = buf.Bytes()
		// A distinct ETag per representation: RFC 9110 wants a strong
		// validator to identify the exact bytes, and these differ.
		a.gzETag = `"` + tag + `-gz"`
	}
	return a
}

type runtimeHandler struct {
	// assets is written once by RuntimeHandler and only read afterwards, so
	// it needs no synchronisation.
	assets map[string]*asset
}

type asset struct {
	body   []byte
	etag   string
	gzBody []byte
	gzETag string
}

// lookup finds a vendored asset by the longest path suffix that matches a
// known file. That is what makes the handler mount-point agnostic for both
// flat names (`alacris.js`) and the UI tree (`ui/theme/index.js`) without
// needing http.StripPrefix — and without remembering names it was asked for.
func (h *runtimeHandler) lookup(reqPath string) *asset {
	name := strings.TrimPrefix(path.Clean("/"+reqPath), "/")
	for name != "" {
		if a, ok := h.assets[name]; ok {
			return a
		}
		slash := strings.IndexByte(name, '/')
		if slash < 0 {
			return nil
		}
		name = name[slash+1:]
	}
	return nil
}

func (h *runtimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a := h.lookup(r.URL.Path)
	if a == nil {
		http.NotFound(w, r)
		return
	}
	head := w.Header()
	head.Set("Content-Type", "text/javascript; charset=utf-8")
	// The body is executable script; nosniff pins the declared type even for a
	// client that would rather guess.
	head.Set("X-Content-Type-Options", "nosniff")

	body, etag := a.body, a.etag
	if a.gzBody != nil {
		head.Set("Vary", "Accept-Encoding")
		if AcceptsGzip(r) {
			body, etag = a.gzBody, a.gzETag
			head.Set("Content-Encoding", "gzip")
		}
	}

	SetCacheHeaders(head, r, etag)
	if MatchETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	head.Set("Content-Length", itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body)
}

// AcceptsGzip reports whether the request says gzip is an acceptable content
// coding. A quality of zero is a refusal, not an acceptance.
func AcceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		coding, q, hasQ := strings.Cut(strings.TrimSpace(part), ";")
		if coding != "gzip" && coding != "*" {
			continue
		}
		if hasQ {
			if v, ok := strings.CutPrefix(strings.TrimSpace(q), "q="); ok && strings.TrimLeft(v, "0.") == "" {
				return false
			}
		}
		return true
	}
	return false
}

// MatchETag reports whether an If-None-Match header matches etag. Unlike a
// substring check it honours the header's real grammar: a comma-separated
// list, an optional W/ weakness prefix, and "*" matching anything.
func MatchETag(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" || etag == "" {
		return false
	}
	for _, part := range strings.Split(ifNoneMatch, ",") {
		part = strings.TrimSpace(part)
		if part == "*" {
			return true
		}
		// A weak comparison is the correct one for If-None-Match.
		if strings.TrimPrefix(part, "W/") == etag {
			return true
		}
	}
	return false
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
	if hasVersionParam(r.URL.RawQuery) {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	h.Set("Cache-Control", "public, no-cache")
}

// hasVersionParam reports whether the query string carries a non-empty v=,
// without building the map r.URL.Query allocates on every call.
func hasVersionParam(rawQuery string) bool {
	for rawQuery != "" {
		var part string
		part, rawQuery, _ = strings.Cut(rawQuery, "&")
		if k, v, _ := strings.Cut(part, "="); k == "v" && v != "" {
			return true
		}
	}
	return false
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

	// NoRecover turns off the live client's automatic recovery. By default,
	// when the patch stream fails permanently — the server restarted, the
	// session expired — the client probes the endpoint until the server
	// answers, reattaches if the session survived, and reloads the page once
	// for a fresh session if it did not. Set NoRecover when the application
	// listens for 'alacris:live' window events and owns that decision itself.
	NoRecover bool

	// Nonce is the CSP nonce for the emitted script tags. When empty, the
	// nonce carried on the context by templ.WithNonce is used.
	Nonce string

	// Version is appended to every asset URL as ?v=, which makes each release
	// a distinct URL and lets the handler cache it for a year instead of
	// revalidating it. Any string that changes with a deploy works: a release
	// tag, a build id, a commit.
	Version string

	// UI loads Alacris UI — every Material Design 3 component, the token
	// system, and applyTheme. The typed wrappers live in the ui subpackage.
	//
	// A zero Theme still applies the default Material scheme (seed #e8ad18,
	// Google Sans Flex, light/dark from the OS). Set Theme to re-skin the
	// page; every component follows because they consume system tokens.
	UI    bool
	Theme Theme
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
	if c.UI {
		// The UI source imports @alacris/core; mapping it onto the same
		// bytes as `alacris` is what keeps one reactive graph on the page.
		imports["@alacris/core"] = imports["alacris"]
		imports["@alacris/ui"] = c.asset(AssetUI)
		imports["@alacris/ui/theme"] = c.asset(AssetUITheme)
		imports["@alacris/ui/motion"] = c.asset("ui/motion/index.js")
		imports["@alacris/ui/tokens"] = c.asset("ui/tokens/index.js")
		imports["@alacris/ui/components/"] = c.base() + "ui/components/"
	}
	for k, v := range c.Imports {
		imports[k] = v
	}
	return imports
}

// Scripts renders the import map, your module entry points, and, when
// configured, the live client.
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

		if c.UI {
			themeJSON, err := json.Marshal(c.Theme.wire())
			if err != nil {
				return err
			}
			b.WriteString(`<script type="module"` + nonceAttr + `>`)
			b.WriteString(`import { applyTheme, setScheme } from '@alacris/ui/theme';`)
			b.WriteString(`import '@alacris/ui';`)
			b.WriteString(`applyTheme(`)
			b.Write(themeJSON)
			b.WriteByte(')')
			b.WriteByte(';')
			if s := c.Theme.scheme(); s == "light" || s == "dark" {
				b.WriteString(`setScheme(` + strconv.Quote(s) + `);`)
			}
			b.WriteString("</script>")
		}

		for _, m := range c.Modules {
			b.WriteString(`<script type="module" src="` + templ.EscapeString(m) + `"` + nonceAttr + `></script>`)
		}

		if c.Live {
			recoverAttr := ""
			if c.NoRecover {
				recoverAttr = ` data-recover="off"`
			}
			b.WriteString(`<script type="module" src="` + templ.EscapeString(c.asset(AssetLive)) + `"` +
				` data-endpoint="` + templ.EscapeString(c.endpoint()) + `"` +
				` data-page="` + templ.EscapeString(c.Page) + `"` + recoverAttr + nonceAttr + `></script>`)
		}

		_, err = io.WriteString(w, b.String())
		return err
	})
}

// Scripts is shorthand for cfg.Scripts().
func Scripts(cfg Config) templ.Component { return cfg.Scripts() }
