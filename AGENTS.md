# AGENTS.md — working on alacris-go

This file is about developing **this module**. It is not the file to give a
project that *uses* the library — that one is [`docs/public/AGENTS.md`][drop-in],
published at <https://bmartel.github.io/alacris-go/AGENTS.md>. The two have
different audiences and different rules, and the fastest way to do damage here
is to follow the wrong one. If you were asked to "add a component" or "wire up
the live layer", you are almost certainly in the wrong repository.

Docs: <https://bmartel.github.io/alacris-go/> · Sibling runtime:
<https://github.com/bmartel/alacris>

[drop-in]: docs/public/AGENTS.md

## What this module is, and what that forbids

1. **It renders alacris custom elements from Go.** A component's shadow
   content is built in the browser by `setup()`. There is no SSR of component
   internals and no declarative shadow DOM. Do not add one, and do not add a
   Go DSL for templates or signals. The `app` nested module is a host for that
   same model (loopback HTTP plus an OS webview), not a second UI toolkit.
2. **It has no dependencies worth the name.** `go.mod` carries `templ` and
   nothing else. Adding a dependency to the root package or `live/` needs a
   reason that survives being asked twice; a dependency in `internal/` or a test
   is ordinary. CGO and the webview library belong in [`app/`](app/), which is
   a nested module so `go test ./...` never links webkit.
3. **`assets/` is someone else's build, vendored byte for byte.** It is not
   edited, not formatted, not reformatted by git ([`.gitattributes`](.gitattributes)
   marks it `-text`), and not "fixed". It is replaced wholesale by
   `internal/vendorjs`.
4. **The generator never guesses.** JavaScript it cannot read as literal data is
   an error, not a best-effort interpretation. A wrong wrapper is worse than a
   failed generate, because it compiles.
5. **A name written into a start tag is structure, not content.** There is no
   escaping that leaves an attribute name meaning the same thing, so names are
   allowlisted and refused at render. Never relax a check in [`names.go`](names.go)
   to make a caller work.
6. **The live layer's two-value auth is load-bearing.** Cookie plus page id,
   both required. Do not collapse them, and do not add a way to drive a session
   from the page id alone.

## The map

```
element.go     the Element builder — props, attrs, slots, rendering
encode.go      Go value -> attribute string, and the limits that raise errors
names.go       tag/attr/element-name validation, inline CSS sanitising
runtime.go     RuntimeHandler, Config, Scripts, RuntimeVersion
theme.go       VarSet and Theme — vars() contracts and applyTheme config
doc.go         package documentation

assets/        VENDORED alacris runtime. Byte-pinned. Never hand-edited.
assets/ui/     VENDORED @alacris/ui source. Same rule.
ui/            GENERATED Alacris UI wrappers. Never hand-edited; internal/genui
internal/genui/  regenerates ui/ from assets/ui
gen/           define() -> typed Go wrappers
  lex.go         a JS lexer that only has to know where tokens end
  parse.go       finds define() calls, reads the props object
  jsdoc.go       @prop/@fires/@slot/@cssprop/@goname — what define() cannot say
  value.go       JS literal -> Go type
  names.go       prop -> attribute, mirroring the runtime's own conversion
  emit.go        the Go source
  manifest.go    the escape hatch for components the scanner cannot read
live/          server-authoritative props over SSE + POST
  server.go      Server, Options, routing, the stream
  session.go     Session, Handle, batching, resync
  cookie.go      the two-value capability model — read this before touching auth
  cors.go        cross-origin, off unless AllowOrigin says otherwise
  patch.go       the wire format
  action.go      Ctx, Bind, On
  assets/live.js the client
cmd/alacris-go/  generate | check | manifest | app init|dev|build|info
internal/appmeta/  alacris.app.json and OS bundles; no webview
internal/vendorjs/  refreshes assets/ from npm
internal/docsgen/   renders every Go example on the docs site
examples/todo/   the example app (a live board) — also the e2e fixture, a `check` target, and `-desktop`
app/             nested module: gated loopback host + OS webview. `-tags desktop` to open a window
e2e/             Playwright, against the real example app
docs/            the Astro site; docs/public/AGENTS.md is the consumer drop-in
```

## The commands

| Command | Purpose |
| --- | --- |
| `go build ./...` | compiles |
| `go vet ./...` | CI runs it; so should you |
| `go test ./...` | the suite |
| `cd app && go test ./...` | host, dialogs, updater, host token; no display |
| `go test -race ./...` | **required** for any change under `live/` |
| `gofmt -l .` | must print nothing — CI fails on any output |
| `go run ./cmd/alacris-go check ./examples/todo/web -o ./examples/todo/ui -strip ala-` | the example's wrappers are current |
| `go run ./internal/docsgen -check` | the site's examples still match the library |
| `go run ./internal/vendorjs -check` | `assets/` still matches npm |
| `go run ./internal/vendorjs -v X.Y.Z` | vendor a new runtime |
| `go run ./internal/vendorjs -ui X.Y.Z` | vendor a new `@alacris/ui` |
| `go run ./internal/genui -check` | the `ui/` wrappers still match `assets/ui` |
| `go run ./internal/genui` | regenerate `github.com/bmartel/alacris-go/ui` |
| `go test -bench . -benchmem ./...` | before and after any performance claim |
| `cd e2e && npx playwright test` | the DOM-identity claims |

CI runs the Go suite on **ubuntu, windows and macos**. Paths, line endings and
temp files have to work on all three; `.gitattributes` normalises everything to
LF on purpose.

## Invariants that look like accidents

These have comments above them explaining why. Read the comment before changing
the code — each one is a bug that already happened.

- **`assets_test.go` pins sha256 of every vendored file.** A failure means the
  vendored runtime drifted: run `go run ./internal/vendorjs`, bump
  `RuntimeVersion` / `UIVersion` in [runtime.go](runtime.go), update the hashes,
  and read the upstream changelog while you are there. Do not "fix" it by
  pasting new hashes without looking at what changed.
- **`runtimeHandler` only ever holds the names it serves.** It used to memoise
  every requested name, including misses, which made a few thousand bogus
  requests into unbounded growth. [`security_test.go`](security_test.go) guards
  this. Any caching added here must be keyed on the fixed asset set.
- **`Options.fill` treats `CookieSameSite == 0` as unset.** `http.SameSiteDefaultMode`
  is 1, not the zero value, and it means "send no attribute at all". The
  attribute is doing real work, so it is always written. Do not simplify.
- **`Element.set` scans linearly** to keep first-set order with last-set value.
  It is `O(n²)` in prop count by design; `BenchmarkElementManyProps` is where a
  regression would show. A map would be faster and would render attributes in a
  different order every run.
- **The generator's lexer is not a parser.** What it cannot interpret it skips
  as balanced source. The parts that must be right are where a token *ends* —
  strings, template literals with nested substitutions, comments, regex
  literals — because getting those wrong makes it read code as data.
  [`gen/parse_test.go`](gen/parse_test.go) is entirely adversarial source; add
  to it when you touch [`gen/lex.go`](gen/lex.go).
- **`gen/names.go` mirrors the runtime's own prop-to-attribute conversion.** If
  it and `define.js` disagree, generated code renders attributes no component is
  listening for. Change both or neither.
- **`sameToken` compares in constant time**, and `newToken` panics rather than
  continue without randomness. Both are deliberate.
- **A boolean prop is always written as `"true"`/`"false"`.** Removing an
  attribute runs `coerce(null, default)`, which yields `false` even when the
  default is `true` — so presence can never mean a boolean. This is why a
  default of `true` generates `*bool`.

## Tests — four layers

1. **Unit** (`*_test.go` beside the code) — encoding, names, parsing, emit.
2. **`security_test.go`** — injection through prop names and slot wrapper tags,
   and the handler's memory bound. Anything that writes into a start tag or
   remembers what it was asked for belongs here.
3. **`live/` with `-race`** — the layer is concurrent by nature. A change here
   that was only run without the race detector has not been tested.
4. **`e2e/`, Playwright against the real example app** — the claims `go test`
   cannot reach: that an update *moves* rows rather than rebuilding them, that
   it leaves focus and a half-typed draft alone, that the capability is a cookie
   and never a URL. `retries: 0` and `workers: 1` are deliberate; a retry would
   hide exactly the flake worth knowing about.

If a node-identity test fails, an `each` has been put inside a conditional.

Two more guard the documentation: `internal/docsgen`'s `-check` (also
`TestDocExamplesAreCurrent`) and the `check` run over `examples/todo`. A third
guards the design system: `go run ./internal/genui -check`.

## Performance claims need numbers

`bench_test.go` and `live/bench_test.go` exist so that "faster" is a measurement.
Run the benchmark before and after and put both in the commit body. A `perf:`
commit with no numbers is not a `perf:` commit.

## Changing the public API

The root package, `live/`, and `ui/` are a published Go module. A tag cannot be
taken back once `proxy.golang.org` has cached it.

1. Change the code and its doc comment — `pkg.go.dev` is the reference.
2. Update or add tests, including `-race` if it is under `live/`.
3. If it changes rendered output or a rule a consumer follows, update the guide
   under `docs/src/content/docs/` **and** [`docs/public/AGENTS.md`][drop-in].
   The drop-in file is the thing agents actually read; leaving it stale is how a
   whole generation of generated code goes wrong at once.
4. If it changes an example's output, `go run ./internal/docsgen`.
5. If it changes generated code, regenerate the example wrappers.
6. If it is breaking, say so with `!` or a `BREAKING CHANGE:` footer.

## Versions and sizes are never typed by hand

- `RuntimeVersion` and `UIVersion` in [runtime.go](runtime.go) are the sources
  of truth for which alacris and `@alacris/ui` builds are vendored.
  `internal/vendorjs` reads them; `assets_test.go` pins the bytes they name.
- The module version is a **git tag**, created by semantic-release. The
  `version` in [package.json](package.json) is never read — that file exists
  only to hold the release tooling, and nothing here is published to npm.
- `CHANGELOG.md` is generated. Do not edit it.

## Commits decide the version

Conventional Commits drive [semantic-release](.releaserc.json). `feat` is a
minor, `fix`/`perf`/`refactor`/`revert` a patch, and `docs`/`test`/`build`/`ci`/
`chore` release nothing.

**While this module is below 1.0.0, a breaking change is a _minor_ bump**, not a
major one — 0.1.2 → 0.2.0, never 1.0.0. That rule is a `releaseRules` override
in `.releaserc.json` with a comment saying to remove it when cutting 1.0.0. It
is not a mistake; leave it alone until then.

## Common mistakes (wrong → right)

| Wrong | Right | Why |
| --- | --- | --- |
| Editing this file for consumer guidance | Edit [`docs/public/AGENTS.md`][drop-in] | Different audience; this one is about the module |
| Hand-editing `assets/*.js` | `go run ./internal/vendorjs -v X.Y.Z` | It is a vendored build; the hashes will catch you |
| Hand-editing `assets/ui/**` or `ui/*_gen.go` | `vendorjs` / `genui` | Both are replaced wholesale |
| Pasting new hashes to make `assets_test.go` pass | Re-vendor and read the changelog | The test is a drift alarm, not a formality |
| Editing `examples/todo/ui/*_gen.go` | Change the component, regenerate | Overwritten, and `check` fails in CI |
| Editing `CHANGELOG.md` | Write a good commit message | It is generated by semantic-release |
| `go test ./live/...` alone | `go test -race ./live/...` | The layer is concurrent; CI runs both |
| Relaxing a check in `names.go` | Fix the caller | Those names go into a start tag unescaped |
| Making the page id sufficient on its own | Cookie **and** page id | The split is what makes a leaked log entry harmless |
| `WriteTimeout` on an `http.Server` in an example | Leave it unset | It cuts every live stream on a timer |
| A `perf:` commit with no benchmark | Numbers before and after | Otherwise it is a `refactor:` |
| Adding a runtime dependency casually | Justify it, or put it in `internal/` | The public packages carry `templ` and nothing else |
| Adding webview or CGO to the root `go.mod` | Put it in `app/` | Nested module, `-tags desktop` |
| A Wails/Fyne/JS `invoke` bridge | `live.On` plus `app.SaveFile` | The live protocol is the interop |
| Bumping a major for a breaking change | `!` and let it cut a minor | Below 1.0.0 by deliberate configuration |

## Verifying your work

Before declaring a task done:

1. `go build ./...`, `go vet ./...`, `gofmt -l .` prints nothing.
2. `go test ./...`, plus `go test -race ./...` if you touched `live/`.
3. `go run ./cmd/alacris-go check ./examples/todo/web -o ./examples/todo/ui -strip ala-`
4. `go run ./internal/docsgen -check`, `go run ./internal/vendorjs -check`, and
   `go run ./internal/genui -check`.
5. If you touched `app/` or `internal/appmeta`: `cd app && go test ./...`.
6. If you touched `live/`, rendering, or the example: `cd e2e && npx playwright test`.
7. If you claimed a performance change: the benchmark, before and after.
8. If you changed a rule a consumer follows: [`docs/public/AGENTS.md`][drop-in]
   and the affected guide say the same thing the code now does.
9. The commit message's type matches what actually changed.

## Reference

- Docs: <https://bmartel.github.io/alacris-go/>
- Go API: <https://pkg.go.dev/github.com/bmartel/alacris-go>
- Wire protocol: <https://bmartel.github.io/alacris-go/reference/wire-protocol/>
- The runtime this module vendors: <https://github.com/bmartel/alacris> — its own
  root `AGENTS.md` covers working on the JavaScript half.
