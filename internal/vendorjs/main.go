// Command vendorjs refreshes the vendored alacris runtime in assets/ from the
// npm registry.
//
// The Go module ships a copy of the JavaScript so that a Go project needs no
// npm at all. A vendored copy drifts unless refreshing it is one command and
// the result is checked, so this program fetches the published tarball and
// TestVendoredAssets pins what it wrote.
//
// Usage:
//
//	go run ./internal/vendorjs             # the version in RuntimeVersion
//	go run ./internal/vendorjs -v 0.3.0    # a specific version
//	go run ./internal/vendorjs -check      # verify assets/ matches, change nothing
//	go run ./internal/vendorjs -bump       # vendor the latest release and rewrite
//	                                       # RuntimeVersion and the pinned hashes
//
// -bump is the whole upgrade in one command, which is what lets a scheduled
// workflow keep the vendored copy in step with upstream: it resolves the
// latest published version (or -v), writes the assets, and updates both
// places that pin them — RuntimeVersion in runtime.go and vendoredHashes in
// assets_test.go — so the only remaining human judgement is reviewing the PR.
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
		version = flag.String("v", "", "alacris version to vendor (default: RuntimeVersion)")
		check   = flag.Bool("check", false, "report differences without writing")
		bump    = flag.Bool("bump", false, "vendor the latest release (or -v) and rewrite RuntimeVersion and the pinned hashes")
		dir     = flag.String("dir", "assets", "output directory")
	)
	flag.Parse()

	if *check && *bump {
		log.Fatal("-check and -bump contradict each other; pick one")
	}

	current, err := runtimeVersion()
	if err != nil {
		log.Fatal(err)
	}
	target := *version
	if target == "" {
		if *bump {
			target = "latest"
		} else {
			target = current
		}
	}

	files, resolved, err := fetch(target)
	if err != nil {
		log.Fatal(err)
	}
	*version = resolved

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
		switch {
		case same:
			fmt.Printf("  ok       %-18s %s\n", name, hex.EncodeToString(sum[:8]))
		case *check:
			changed++
			fmt.Printf("  DIFFERS  %-18s %s\n", name, hex.EncodeToString(sum[:8]))
		default:
			if err := os.WriteFile(dst, body, 0o644); err != nil {
				log.Fatal(err)
			}
			changed++
			fmt.Printf("  wrote    %-18s %s\n", name, hex.EncodeToString(sum[:8]))
		}
	}

	if *check && changed > 0 {
		log.Fatalf("%d vendored file(s) differ from alacris@%s", changed, *version)
	}

	if *bump {
		if err := rewritePins(*version, files); err != nil {
			log.Fatal(err)
		}
		if *version == current && changed == 0 {
			fmt.Printf("\nalacris@%s is already vendored and current\n", current)
		} else {
			fmt.Printf("\nbumped alacris %s -> %s; RuntimeVersion and assets_test.go updated\n", current, *version)
		}
		return
	}
	if !*check && changed > 0 {
		fmt.Printf("\nvendored alacris@%s — update RuntimeVersion and the hashes in assets_test.go\n", *version)
	}
}

// rewritePins updates the two places that pin the vendored bytes, so a bump
// is one command rather than a command plus two hand edits that can be
// forgotten or mistyped — mistyped being the dangerous one, since a hash
// copied wrong is a pin on nothing.
func rewritePins(version string, files map[string][]byte) error {
	if err := rewrite("runtime.go", versionRE, `RuntimeVersion = "`+version+`"`); err != nil {
		return err
	}
	for name, body := range files {
		if name == "LICENSE.alacris" {
			continue
		}
		sum := sha256.Sum256(body)
		re, err := regexp.Compile(`"` + regexp.QuoteMeta(name) + `":\s*"[0-9a-f]{64}"`)
		if err != nil {
			return err
		}
		repl := `"` + name + `": "` + hex.EncodeToString(sum[:]) + `"`
		if err := rewrite("assets_test.go", re, repl); err != nil {
			return err
		}
	}
	// The hash replacements do not preserve the map's alignment; leave the
	// file the way gofmt would, so the bump does not trip the format check.
	return gofmtFile("assets_test.go")
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

var versionRE = regexp.MustCompile(`RuntimeVersion = "([^"]+)"`)

func runtimeVersion() (string, error) {
	src, err := os.ReadFile("runtime.go")
	if err != nil {
		return "", fmt.Errorf("reading runtime.go for RuntimeVersion: %w", err)
	}
	m := versionRE.FindSubmatch(src)
	if m == nil {
		return "", fmt.Errorf("RuntimeVersion not found in runtime.go")
	}
	return string(m[1]), nil
}

// fetch downloads and verifies one published version. version may be a
// dist-tag like "latest"; resolved is always the concrete version number the
// registry answered with.
func fetch(version string) (files map[string][]byte, resolved string, err error) {
	client := &http.Client{Timeout: 60 * time.Second}

	metaURL := "https://registry.npmjs.org/alacris/" + version
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
		return nil, "", fmt.Errorf("no tarball for alacris@%s", version)
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
		return nil, "", fmt.Errorf("alacris@%s: %w", version, err)
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
		name, ok := wanted[path.Clean(hdr.Name)]
		if !ok {
			continue
		}
		// The published tarball is small; a limit keeps a hostile one from
		// exhausting memory.
		body, err := io.ReadAll(io.LimitReader(tr, 4<<20))
		if err != nil {
			return nil, "", fmt.Errorf("reading %s: %w", hdr.Name, err)
		}
		out[name] = body
	}

	var missing []string
	for _, name := range wanted {
		if _, ok := out[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, "", fmt.Errorf("alacris@%s is missing %s", version, strings.Join(missing, ", "))
	}
	return out, resolved, nil
}
