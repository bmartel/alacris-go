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
package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
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
		dir     = flag.String("dir", "assets", "output directory")
	)
	flag.Parse()

	if *version == "" {
		v, err := runtimeVersion()
		if err != nil {
			log.Fatal(err)
		}
		*version = v
	}

	files, err := fetch(*version)
	if err != nil {
		log.Fatal(err)
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
	if !*check && changed > 0 {
		fmt.Printf("\nvendored alacris@%s — update RuntimeVersion and the hashes in assets_test.go\n", *version)
	}
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

func fetch(version string) (map[string][]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}

	metaURL := "https://registry.npmjs.org/alacris/" + version
	resp, err := client.Get(metaURL)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", metaURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: %s", metaURL, resp.Status)
	}
	var meta struct {
		Dist struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decoding registry metadata: %w", err)
	}
	if meta.Dist.Tarball == "" {
		return nil, fmt.Errorf("no tarball for alacris@%s", version)
	}

	tarResp, err := client.Get(meta.Dist.Tarball)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", meta.Dist.Tarball, err)
	}
	defer tarResp.Body.Close()
	if tarResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: %s", meta.Dist.Tarball, tarResp.Status)
	}

	gz, err := gzip.NewReader(tarResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading tarball: %w", err)
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
			return nil, fmt.Errorf("reading tarball: %w", err)
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
			return nil, fmt.Errorf("reading %s: %w", hdr.Name, err)
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
		return nil, fmt.Errorf("alacris@%s is missing %s", version, strings.Join(missing, ", "))
	}
	return out, nil
}
