// Command vendorjs refreshes the vendored alacris runtime and @alacris/ui
// sources in assets/ from the npm registry.
//
// The Go module ships a copy of the JavaScript so that a Go project needs no
// npm at all. A vendored copy drifts unless refreshing it is one command and
// the result is checked, so this program fetches the published tarball and
// TestVendoredAssets pins what it wrote.
//
// Usage:
//
//	go run ./internal/vendorjs             # versions in RuntimeVersion / UIVersion
//	go run ./internal/vendorjs -v 0.3.0    # a specific alacris version
//	go run ./internal/vendorjs -ui 0.2.0   # a specific @alacris/ui version
//	go run ./internal/vendorjs -check      # verify assets/ matches, change nothing
//	go run ./internal/vendorjs -bump       # vendor the latest releases and rewrite
//	                                       # RuntimeVersion, UIVersion and the hashes
//
// -bump is the whole upgrade in one command, which is what lets a scheduled
// workflow keep the vendored copy in step with upstream: it resolves the
// latest published versions (or -v / -ui), writes the assets, and updates the
// pins — so the only remaining human judgement is reviewing the PR.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// wanted maps a path inside the npm tarball to its name in assets/.
var wanted = map[string]string{
	"package/dist/alacris.js":     "alacris.js",
	"package/dist/alacris.dev.js": "alacris.dev.js",
	"package/dist/store.js":       "store.js",
	"package/dist/context.js":     "context.js",
	"package/dist/signal.js":      "signal.js",
	"package/LICENSE":             "LICENSE.alacris",
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("vendorjs: ")

	var (
		version   = flag.String("v", "", "alacris version to vendor (default: RuntimeVersion)")
		uiVersion = flag.String("ui", "", "@alacris/ui version to vendor (default: UIVersion)")
		check     = flag.Bool("check", false, "report differences without writing")
		bump      = flag.Bool("bump", false, "vendor the latest releases and rewrite version pins and hashes")
		dir       = flag.String("dir", "assets", "output directory")
	)
	flag.Parse()

	if *check && *bump {
		log.Fatal("-check and -bump contradict each other; pick one")
	}

	current, err := pinnedVersion(versionRE, "RuntimeVersion")
	if err != nil {
		log.Fatal(err)
	}
	currentUI, err := pinnedVersion(uiVersionRE, "UIVersion")
	if err != nil {
		log.Fatal(err)
	}

	coreTarget := *version
	if coreTarget == "" {
		if *bump {
			coreTarget = "latest"
		} else {
			coreTarget = current
		}
	}
	uiTarget := *uiVersion
	if uiTarget == "" {
		if *bump {
			uiTarget = "latest"
		} else {
			uiTarget = currentUI
		}
	}

	coreFiles, resolved, err := fetch("alacris", coreTarget, corePick, wantedNames())
	if err != nil {
		log.Fatal(err)
	}
	uiFiles, uiResolved, err := fetch("@alacris/ui", uiTarget, uiPick, []string{"ui/index.js", "LICENSE.alacris-ui"})
	if err != nil {
		log.Fatal(err)
	}

	files := map[string][]byte{}
	for k, v := range coreFiles {
		files[k] = v
	}
	for k, v := range uiFiles {
		files[k] = v
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	changed := 0
	for _, name := range names {
		body := files[name]
		dst := filepath.Join(*dir, name)
		old, err := os.ReadFile(dst)
		same := err == nil && string(old) == string(body)
		sum := sha256.Sum256(body)
		label := name
		if len(label) > 40 {
			label = label[:37] + "..."
		}
		switch {
		case same:
			fmt.Printf("  ok       %-40s %s\n", label, hex.EncodeToString(sum[:8]))
		case *check:
			changed++
			fmt.Printf("  DIFFERS  %-40s %s\n", label, hex.EncodeToString(sum[:8]))
		default:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				log.Fatal(err)
			}
			if err := os.WriteFile(dst, body, 0o644); err != nil {
				log.Fatal(err)
			}
			changed++
			fmt.Printf("  wrote    %-40s %s\n", label, hex.EncodeToString(sum[:8]))
		}
	}

	if !*check {
		if err := pruneStale(*dir, uiFiles); err != nil {
			log.Fatal(err)
		}
	}

	if *check && changed > 0 {
		log.Fatalf("%d vendored file(s) differ from alacris@%s / @alacris/ui@%s", changed, resolved, uiResolved)
	}

	if *bump {
		if err := rewritePins(resolved, uiResolved, files); err != nil {
			log.Fatal(err)
		}
		if resolved == current && uiResolved == currentUI && changed == 0 {
			fmt.Printf("\nalacris@%s and @alacris/ui@%s are already vendored and current\n", current, currentUI)
		} else {
			fmt.Printf("\nbumped alacris %s -> %s, @alacris/ui %s -> %s; pins updated\n",
				current, resolved, currentUI, uiResolved)
		}
		return
	}
	if !*check && changed > 0 {
		if err := rewriteHashMap(files); err != nil {
			log.Fatal(err)
		}
		if err := gofmtFile("assets_test.go"); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\nvendored alacris@%s and @alacris/ui@%s; hashes updated\n", resolved, uiResolved)
	}
}

func wantedNames() []string {
	var names []string
	for _, n := range wanted {
		names = append(names, n)
	}
	return names
}

func corePick(p string) (string, bool) {
	name, ok := wanted[p]
	return name, ok
}

func uiPick(p string) (string, bool) {
	p = path.Clean(p)
	if p == "package/LICENSE" {
		return "LICENSE.alacris-ui", true
	}
	const prefix = "package/src/"
	if !strings.HasPrefix(p, prefix) {
		return "", false
	}
	rel := strings.TrimPrefix(p, prefix)
	if rel == "" {
		return "", false
	}
	return path.Join("ui", rel), true
}

func pruneStale(assetsDir string, uiFiles map[string][]byte) error {
	uiDir := filepath.Join(assetsDir, "ui")
	keep := map[string]bool{}
	for name := range uiFiles {
		if rest, ok := strings.CutPrefix(name, "ui/"); ok {
			keep[filepath.FromSlash(rest)] = true
		}
	}
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(uiDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(uiDir, p)
		if err != nil {
			return err
		}
		if keep[rel] {
			return nil
		}
		fmt.Printf("  removed  %s\n", filepath.ToSlash(filepath.Join("ui", rel)))
		return os.Remove(p)
	})
}

// rewritePins updates the places that pin the vendored bytes, so a bump is
// one command rather than a command plus hand edits that can be forgotten
// or mistyped — mistyped being the dangerous one, since a hash copied wrong
// is a pin on nothing.
func rewritePins(version, uiVersion string, files map[string][]byte) error {
	if err := rewrite("runtime.go", versionRE, `RuntimeVersion = "`+version+`"`); err != nil {
		return err
	}
	if err := rewrite("runtime.go", uiVersionRE, `UIVersion = "`+uiVersion+`"`); err != nil {
		return err
	}
	if err := rewriteHashMap(files); err != nil {
		return err
	}
	return gofmtFile("assets_test.go")
}

func rewriteHashMap(files map[string][]byte) error {
	var b strings.Builder
	b.WriteString("var vendoredHashes = map[string]string{\n")
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, "LICENSE") {
			continue
		}
		if !strings.HasSuffix(name, ".js") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sum := sha256.Sum256(files[name])
		fmt.Fprintf(&b, "\t%q: %q,\n", name, hex.EncodeToString(sum[:]))
	}
	b.WriteString("}")
	re := regexp.MustCompile(`var vendoredHashes = map\[string\]string\{(?s:.*?)\}`)
	return rewrite("assets_test.go", re, b.String())
}

func gofmtFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := format.Source(src)
	if err != nil {
		return fmt.Errorf("%s is no longer valid Go after rewriting: %w", path, err)
	}
	return os.WriteFile(path, out, 0o644)
}

// rewrite replaces the single match of re in path. No match is an error: a
// pin that cannot be found is a pin that is not being updated.
func rewrite(path string, re *regexp.Regexp, repl string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !re.Match(src) {
		return fmt.Errorf("%s: nothing matches %s; the pin moved and this tool needs updating", path, re)
	}
	return os.WriteFile(path, re.ReplaceAll(src, []byte(repl)), 0o644)
}

// verifyDist checks the downloaded tarball against the hash the registry's
// metadata declared for it: dist.integrity (an SRI sha512) when present,
// dist.shasum (hex sha1) otherwise. No hash at all is an error — vendoring
// unverifiable bytes silently is how a bad fetch becomes a pinned asset.
func verifyDist(raw []byte, integrity, shasum string) error {
	if rest, ok := strings.CutPrefix(integrity, "sha512-"); ok {
		sum := sha512.Sum512(raw)
		if got := base64.StdEncoding.EncodeToString(sum[:]); got != rest {
			return fmt.Errorf("tarball sha512 %s does not match the registry's dist.integrity %s", got, rest)
		}
		return nil
	}
	if shasum != "" {
		sum := sha1.Sum(raw)
		if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, shasum) {
			return fmt.Errorf("tarball sha1 %s does not match the registry's dist.shasum %s", got, shasum)
		}
		return nil
	}
	return fmt.Errorf("the registry offered no dist.integrity or dist.shasum to verify the tarball against")
}

var (
	versionRE   = regexp.MustCompile(`RuntimeVersion = "([^"]+)"`)
	uiVersionRE = regexp.MustCompile(`UIVersion = "([^"]+)"`)
)

func pinnedVersion(re *regexp.Regexp, name string) (string, error) {
	src, err := os.ReadFile("runtime.go")
	if err != nil {
		return "", fmt.Errorf("reading runtime.go for %s: %w", name, err)
	}
	m := re.FindSubmatch(src)
	if m == nil {
		return "", fmt.Errorf("%s not found in runtime.go", name)
	}
	return string(m[1]), nil
}

func runtimeVersion() (string, error) { return pinnedVersion(versionRE, "RuntimeVersion") }

// fetch downloads and verifies one published version. version may be a
// dist-tag like "latest"; resolved is always the concrete version number the
// registry answered with.
func fetch(pkg, version string, pick func(string) (string, bool), required []string) (files map[string][]byte, resolved string, err error) {
	client := &http.Client{Timeout: 60 * time.Second}

	metaURL := "https://registry.npmjs.org/" + pkg + "/" + version
	resp, err := client.Get(metaURL)
	if err != nil {
		return nil, "", fmt.Errorf("fetching %s: %w", metaURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetching %s: %s", metaURL, resp.Status)
	}
	var meta struct {
		Version string `json:"version"`
		Dist    struct {
			Tarball   string `json:"tarball"`
			Integrity string `json:"integrity"`
			Shasum    string `json:"shasum"`
		} `json:"dist"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, "", fmt.Errorf("decoding registry metadata: %w", err)
	}
	if meta.Dist.Tarball == "" {
		return nil, "", fmt.Errorf("no tarball for %s@%s", pkg, version)
	}
	resolved = meta.Version
	if resolved == "" {
		resolved = version
	}

	tarResp, err := client.Get(meta.Dist.Tarball)
	if err != nil {
		return nil, "", fmt.Errorf("fetching %s: %w", meta.Dist.Tarball, err)
	}
	defer tarResp.Body.Close()
	if tarResp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetching %s: %s", meta.Dist.Tarball, tarResp.Status)
	}

	// The registry publishes the tarball's hash alongside its URL. Checking it
	// closes the gap the version pin cannot: a bad fetch would otherwise be
	// vendored, printed, and then blessed when its hashes are copied into
	// assets_test.go.
	raw, err := io.ReadAll(io.LimitReader(tarResp.Body, 64<<20))
	if err != nil {
		return nil, "", fmt.Errorf("reading tarball: %w", err)
	}
	if err := verifyDist(raw, meta.Dist.Integrity, meta.Dist.Shasum); err != nil {
		return nil, "", fmt.Errorf("%s@%s: %w", pkg, version, err)
	}

	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, "", fmt.Errorf("reading tarball: %w", err)
	}
	defer gz.Close()

	out := map[string][]byte{}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("reading tarball: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name, ok := pick(path.Clean(hdr.Name))
		if !ok {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(tr, 4<<20))
		if err != nil {
			return nil, "", fmt.Errorf("reading %s: %w", hdr.Name, err)
		}
		out[name] = body
	}

	var missing []string
	for _, name := range required {
		if _, ok := out[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, "", fmt.Errorf("%s@%s is missing %s", pkg, version, strings.Join(missing, ", "))
	}
	return out, resolved, nil
}
