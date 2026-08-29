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
	"alacris.dev.js":                        "5c4f2fcd09fcea011885512eb63447bd8da6a933f3fa6a7d60bb349abfd29d7a",
	"alacris.js":                            "ac091f21ea6823c05fa1993444092b60e707aefe194d0640a1b89e891d8fcef6",
	"context.js":                            "c571d617572b608b4e2c3f2d0b0e36221e9afee76e966bd1c2bc7614515f1e91",
	"signal.js":                             "a37194243c8fcaae18ca5a204f87bd5bfec5aabe9dd701b4d40f06e4ca8bdcac",
	"store.js":                              "d9cc52ee9f6f9ac251f2c94e54fee65de5695b9656eff1c48fa864f4710fb27d",
	"ui/components/base.js":                 "f884d7bd7368828529c9df4aefa7d5d739f2e37257d048bf8897c34409758eb0",
	"ui/components/ui-accordion-item.js":    "aec6dc3c3d059ba4690c0bcd6fee7d357ea68d6d58f2b29252cb4b1bf7d894fc",
	"ui/components/ui-accordion.js":         "030b507720c34e5b10e65a86983834af938b9d92e5e834c22926b4f98399e634",
	"ui/components/ui-alert.js":             "a8448d2a672e929cb9503eaaa5844afd02ee9f6f49fe66642a6e9a2c827ad1fd",
	"ui/components/ui-app-bar.js":           "bf80e4f0582ec85c6e87a9bf10a8d253af2bcfd8239b67fc9af37511c81261fa",
	"ui/components/ui-autocomplete.js":      "956fc2c07cc2a48d19a6cdf77796aa3b80a1a746834ba8960331313999dd6b18",
	"ui/components/ui-avatar.js":            "32847d2873fd3ebdb9b1f1c840b379e9144b6309e476b2d789970db380576d77",
	"ui/components/ui-backdrop.js":          "1472d85b227953b2f8b1131fca02a83345b5f4782113a654093b4036692ce9d4",
	"ui/components/ui-badge.js":             "64af359e63e7b5620b6b759710569afb1f0f7b3aea331e149c6eb88f6bf179b1",
	"ui/components/ui-bottom-app-bar.js":    "48f42a267d77a33c5f120ca1178c3835d9d434a2925694bd7b0f081ca1046b96",
	"ui/components/ui-bottom-nav.js":        "6bf30a4b7894fb5d773ddab2e82892bb3453480153cc15a131266b3780742ef2",
	"ui/components/ui-breadcrumbs.js":       "b740053092944aa20429321cec3e10b84fafecb0875794763819d39a72c37bcc",
	"ui/components/ui-button-group.js":      "c2c15e490a7eb1fa6b1e9f092d0546d4c3d51a0b072b8c2a0264c040e987c018",
	"ui/components/ui-button.js":            "e07ef2ca01f56f4cfc95e6973ccdb90d089b8628d6032e00502e39260ef79c82",
	"ui/components/ui-card.js":              "3d2b56437e1779d53dbfce012f7d66ae3b381d7a1ac211b1f638282b7e36ad1b",
	"ui/components/ui-carousel-item.js":     "60853e87b599777b3dc03318ded8ca86324a7d5f76f8d084ca8b7b847ff7b609",
	"ui/components/ui-carousel.js":          "4a12471d07960721606ef4a27df76234b582a60ebdc4ab49446fc76a41c46e13",
	"ui/components/ui-checkbox.js":          "65e57fbc396428945ce66ed58f610e4511370ced1da1e27aa45ce4b87b33af88",
	"ui/components/ui-chip-set.js":          "94bfbcd5dd361589ce8a7868117586092b439a1c868289ea8d67a8d169aa3ab8",
	"ui/components/ui-chip.js":              "112fe6a0aac59c8e65887ed6f58a57ad2875737f79cdb2bc8f6005ac98b0f458",
	"ui/components/ui-container.js":         "3be7cee7479ab1b9c5a0f6979ce6fcbb6399b836d8032324f7b7a31b8414cef3",
	"ui/components/ui-date-picker.js":       "3097436d0306d650312d2216599fcea41d4be64c372bd6b8f3ca0968c8153383",
	"ui/components/ui-dialog.js":            "555b95e6f0055d99babd37df1a7d9ecb386c2b4aaba7c6abd31fc6c60321de20",
	"ui/components/ui-divider.js":           "c52f23831cd47da018bfaf4e1268af14266bbf2f149cc655ba78f136539e1022",
	"ui/components/ui-drawer.js":            "ac13e54b7409d8cd0cb99bdbc4b7d358b020ab8b44457450f768f4d8ee2a9c87",
	"ui/components/ui-fab-menu.js":          "aea824ad27239ec8356d4244d8bfc3330ec0db4e0b82d4f61baec316db776045",
	"ui/components/ui-fab.js":               "bdd083c5451b9ed1edb2b628b7b8fc0869a1f4314b11d863b53164fded096298",
	"ui/components/ui-icon-button.js":       "275178059e599ccbf0d37fde026416fcc9300f1b4d403b127ad57e038c8db835",
	"ui/components/ui-icon.js":              "aac25f11a016050f253beb61c251737e160b72c42bf1f6fae1239665844246f3",
	"ui/components/ui-list-item.js":         "dc9aaad0c2a1eaffd8796c541531af20e7752306cb316ce96dc998db10451583",
	"ui/components/ui-list.js":              "d7bb4fab1a45bb2e4ed7c28fc8b8e19e39829aea69b5d9513d61901742901b76",
	"ui/components/ui-loading-indicator.js": "28b4a5f5fed5644f9c570bacd467a75e59817670c455092a058d405c07740a6b",
	"ui/components/ui-menu-item.js":         "a5ef7378cc055c3098468cafbc6c954d3e658a3c6cfa6625c43cd696c47e4a0f",
	"ui/components/ui-menu.js":              "205ec4e848a62eafae6dcbd214451c9296c23b602dffd0aaec839b110b4f13eb",
	"ui/components/ui-nav-item.js":          "f26bb779e5ebf15740d36e8c018285f56b401d03f3efcdf66f6ff2463ac04d40",
	"ui/components/ui-nav-rail.js":          "57dd3b75726dd3c608d2273958ed1bcdf98939910852f7e74f4fe433b1312740",
	"ui/components/ui-option.js":            "d017efbfc3a401ae25394375ac327a60cfec499c2463122f57c7906e35e7259b",
	"ui/components/ui-pagination.js":        "ce1ff348001185f0a531cd85f6958e432a67a1f4bd7b28ce10b3a76adc801512",
	"ui/components/ui-progress.js":          "28b181849133c9210b25673052b028413a5b190a994f0f097ab3e51d18a653ce",
	"ui/components/ui-radio-group.js":       "f19da2645da85f6a9337907f0d0945665b4b009ef112c17f7e0c1cb70c72ed37",
	"ui/components/ui-radio.js":             "10e702aa67d10c422ed659165d46a83ca161871bfc8e461a45790be6959cdf35",
	"ui/components/ui-rating.js":            "7c328f4b404c7b1341abe82bc618b534068ec0dd8a28cca3f9ace47ec58fbdb3",
	"ui/components/ui-search.js":            "f4af4963986b8f37a0948c901bddaf217d2c3f4497f30ac0b6fa406d297f44fa",
	"ui/components/ui-select.js":            "e09148d6272b10aaa78d339f26bf5487212043ad7d624994d52e3988141c299f",
	"ui/components/ui-sheet.js":             "6f566bd3a98f11563f620cdb324cebc097b56d2a4c3419c919b844506f142d32",
	"ui/components/ui-side-sheet.js":        "2b113cbe75a51ef40cc2e82a7a1839989fc2400360f70947826d3af7dd2fb40d",
	"ui/components/ui-skeleton.js":          "735ffb81b95bc3810fc52b90dab1a78db34ad0506e4ee1dd5c640435d1458e11",
	"ui/components/ui-slider.js":            "a402424c11fc72295552a21b13c452b52f3261b7d79cc83422d3eff4e2ca00a1",
	"ui/components/ui-snackbar.js":          "69fe5fc14c831138f718cb330b99a7e30963cca28beb1da50afb218daaa74e4a",
	"ui/components/ui-spinner.js":           "45e0fe35b969f95d973629d7e9105f014410bfc7576aff77eefc728f4ba15a78",
	"ui/components/ui-split-button.js":      "585cb833613d7640003943d42ab8b224b4c2e9c4264e47ea8aad0c74e09aee24",
	"ui/components/ui-stack.js":             "368aa33a5756207f104525925244f2ed0ac55ed3ed5a81ab0fb424285cd933d6",
	"ui/components/ui-step.js":              "24a40f1183b1b97ac6e8ee7e71cdbd5a8d7d148473c19907b4fbff0c02a79f3c",
	"ui/components/ui-stepper.js":           "4c89b272d1befa189d71ca56b8a32c038e0307dcc1b04b19f1915b404a6be452",
	"ui/components/ui-surface.js":           "a2c68b9f4fe9da95f02356dac7a51e738dc77eb0e04788c905a52b6a41d32106",
	"ui/components/ui-switch.js":            "f63c4ba401451c35d1341e1d8120f4cb30fc6253c03b730c90fb6f0ed6424c45",
	"ui/components/ui-tab-panel.js":         "f9806de8d5ebccdaddc9ad91a7a662b1f3631a82462391dcb4e6a64f4ca91951",
	"ui/components/ui-tab.js":               "f8afd87bebfa5e6e0c286c765907d9515f75f46139b8d3ff74608086e15da0ee",
	"ui/components/ui-table-footer.js":      "6afcc867e269202f1051dec77b4ae53d81f905d362fdbcf37a0056f87ecdcc6c",
	"ui/components/ui-table-toolbar.js":     "d471ed2d3c4bd1718492403a575c82a63b54fe422bace401221966d883d6125f",
	"ui/components/ui-table.js":             "300188ec33bb2f7cb6d82f9e311d536b057b2e12eea93ba1fb515ec072ac748b",
	"ui/components/ui-tabs.js":              "c07f37069943e08a22665844efe44c8547e59394bf5934cbe86b9206ee879fb0",
	"ui/components/ui-text-field.js":        "6cf1eba6718f51c6cd504b4ea6b7ac8a49900968170599f15e71dbeb75caf625",
	"ui/components/ui-text.js":              "307bd7f81c6d7af3b8e82f5b82f2e5c45b60f13f7b453a214c712203f6b4d148",
	"ui/components/ui-time-picker.js":       "0f7ba600cefa8f82f6d5e923ba05ff0d0a0f9b493326c4afacf50188936c17a4",
	"ui/components/ui-toggle-button.js":     "7e141b6ddee57987acfb9909fa2d9c2ce9b8890af42e5e394d496a7c0f3a2423",
	"ui/components/ui-toggle-group.js":      "ca8360474d7aabbc38a151ffc9a6fa70739f99a7b768d6432088cd20a176f0af",
	"ui/components/ui-toolbar.js":           "f2a3d95df7a9aba59f8918238a178839b8387cd363c0457c3a2da699b0260095",
	"ui/components/ui-tooltip.js":           "f67d149bb2255fa1ba3345f3bfb2d74a71bf75e9359031199d4f5ca4cb598151",
	"ui/index.js":                           "98f38af7c20e5d71d7a976ed7e1e53da50561dcce51a51fc718104cfcf97525f",
	"ui/motion/animate.js":                  "fdba67a645bf65bd88c5aa813e58c0ac28606e86a1c49f9ea67c19d4479a51eb",
	"ui/motion/flip.js":                     "a7a31a5996d2753eb3b40d943d8a5e9bda5f27d4d4c0d4500c32b91984a4951a",
	"ui/motion/index.js":                    "a1ab106432a0e2ae3a95d4769fe81108e7906cbf9bc22e15097411c3a3a48e2e",
	"ui/motion/presence.js":                 "8543a02778b488c65ee044ff1c63b734c8ab3b60f1fc5621a5ccbad2af43e516",
	"ui/motion/ripple.js":                   "755137114ac45cc7648a4020d48cb636421babc325736e5186d3c0f1c2922421",
	"ui/theme/apply-theme.js":               "2cd276f7136d666ca36f8f51db03aea345a131ff68884f485d90a43839dc2905",
	"ui/theme/create-theme.js":              "e9767bd5b2687244d24a9ce5da2d3e609bbab717e8437ebecdfebf95a4f0ae7a",
	"ui/theme/index.js":                     "3850868ed1ccd9529acbebaf66bc383acb9f5d14d1bf5e878078b1f3944b4971",
	"ui/tokens/color.js":                    "0e4b756cabfd0eb65d6177d719fb6d15befb79dddc853a401288365a1781c49f",
	"ui/tokens/index.js":                    "e9266023bdfd2c2fda496e4f49863fe7f7e649e55add00b5613cde04ccee6b8c",
	"ui/tokens/sys.js":                      "7728a0ee45c799a32111db8e69bbee8f2dd932ed1731f8d4c0eb27d34569a2ed",
	"ui/tokens/system.js":                   "e80faed535d9e90f25d12b6a4620310d6043aa513d51cbfe2760aad02a64ae76",
	"ui/tokens/typography.js":               "a9a356069c2444fadf53638b9ae0036a77e09b81ccaebf0ea087384619e73325",
	"ui/util/focus.js":                      "6229cb4a9d50ecdca3247715ea25b2c31d67d37b91a924c97f45492fb761437f",
	"ui/util/form.js":                       "4be83494d5bc1338dd0483bd0d541e1fa438bd7a6b38cb4464759fd01e191e70",
	"ui/util/icons.js":                      "a96fee11613a54df00534f029ba97db0b777a97189369b6885cf61ee7c0aa955",
	"ui/util/keys.js":                       "3816a5bb6e11f0dd5d84d2f8343cda076da09c314a03229003c13ef7377cad9e",
	"ui/util/popup.js":                      "fe1e8537fa980578245914a805f895f37363bdf15c0610d5ebccb8d9dc9d4b31",
	"ui/util/position.js":                   "b4d60c1ce73261f19bb0ce3d7a92e6772bdd8e6113436cab33ddeb38730bd130",
	"ui/util/table.js":                      "56f6c4ea5ee99729064953f0fd524cc7640cd1299ecb974e111a4a86de3566ae",
}

func TestVendoredAssets(t *testing.T) {
	assets := Assets()
	seen := map[string]bool{}
	err := fs.WalkDir(assets, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(name, ".js") {
			return nil
		}
		seen[name] = true
		want, ok := vendoredHashes[name]
		if !ok {
			t.Errorf("%s: present in assets/ but not in vendoredHashes; run: go run ./internal/vendorjs", name)
			return nil
		}
		body, err := fs.ReadFile(assets, name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			return nil
		}
		sum := sha256.Sum256(body)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("%s: hash %s, want %s\n\trun: go run ./internal/vendorjs", name, got, want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for name := range vendoredHashes {
		if !seen[name] {
			t.Errorf("%s: hashed but missing from assets/", name)
		}
	}

	for _, lic := range []string{"LICENSE.alacris", "LICENSE.alacris-ui"} {
		if _, err := fs.ReadFile(assets, lic); err != nil {
			t.Errorf("the vendored packages must ship their licenses: %s: %v", lic, err)
		}
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

	t.Run("serves nested UI modules", func(t *testing.T) {
		h := RuntimeHandler()
		for _, path := range []string{
			"/_alacris/ui/index.js",
			"/static/ui/index.js",
			"/_alacris/ui/theme/index.js",
		} {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK {
				t.Errorf("GET %s: status %d", path, rec.Code)
			}
		}
		// Two different index.js files must not collapse to basename.
		idx := httptest.NewRecorder()
		h.ServeHTTP(idx, httptest.NewRequest(http.MethodGet, "/_alacris/ui/index.js", nil))
		theme := httptest.NewRecorder()
		h.ServeHTTP(theme, httptest.NewRequest(http.MethodGet, "/_alacris/ui/theme/index.js", nil))
		if idx.Body.String() == theme.Body.String() {
			t.Error("ui/index.js and ui/theme/index.js served the same bytes")
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
