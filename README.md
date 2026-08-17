<div align="center">

<img src="https://raw.githubusercontent.com/bmartel/alacris-go/main/docs/public/logo.png" alt="" width="88" height="88">

# alacris-go

**Build [alacris](https://github.com/bmartel/alacris) web components from Go and [templ](https://templ.guide).**

Typed wrappers generated from your `define()` calls · the runtime served from Go, no npm · server-driven props with no HTML on the wire

[![CI](https://github.com/bmartel/alacris-go/actions/workflows/ci.yml/badge.svg)](https://github.com/bmartel/alacris-go/actions/workflows/ci.yml)
[![Docs](https://github.com/bmartel/alacris-go/actions/workflows/docs.yml/badge.svg)](https://bmartel.github.io/alacris-go/)
[![Go Reference](https://pkg.go.dev/badge/github.com/bmartel/alacris-go.svg)](https://pkg.go.dev/github.com/bmartel/alacris-go)
[![license](https://img.shields.io/badge/license-MIT-green)](LICENSE)

**[Documentation →](https://bmartel.github.io/alacris-go/)** · **[Go API →](https://pkg.go.dev/github.com/bmartel/alacris-go)**

</div>

```go
@ui.Board(ui.BoardProps{Items: items}).
    ID("board").
    On(ui.BoardEventAdd, "add-card")
```

```go
live.On(srv, "add-card", func(c *live.Ctx, d ui.BoardAddDetail) error {
    list.Add(d.Text, d.Column)
    c.Session.Element("board").Set("items", list.Items())   // one property write
    return nil
})
```

A prop is a signal. A server change is one DOM property write and one node
update: no HTML over the wire, and focus, scroll position and typed text stay
put.

Those names are the example app's: `examples/todo` generates wrappers into
package `ui` and aliases the design system as `m3`. In a new project, generate
app wrappers to `./internal/components` and import
[`github.com/bmartel/alacris-go/ui`](https://pkg.go.dev/github.com/bmartel/alacris-go/ui)
as `ui`.

## Install

```bash
go get github.com/bmartel/alacris-go
```

The alacris runtime is vendored into the module and served from Go, so a Go
project needs no npm. Alacris UI (sixty-eight Material Design 3 components) is
vendored the same way. `Config.UI` turns it on; the typed wrappers live in
[`github.com/bmartel/alacris-go/ui`](https://pkg.go.dev/github.com/bmartel/alacris-go/ui).

## What the server can and cannot render

A component's shadow content is produced by its `setup()` function when the
element connects, **in the browser**. There is no server-side rendering of
component internals and no declarative shadow DOM. What Go renders is the
element itself:

```html
<user-card name="Ada" age="36" tags="[&quot;math&quot;,&quot;code&quot;]">
  <h3 slot="title">Ada Lovelace</h3>   <!-- light DOM: in the first paint -->
</user-card>
```

Slot content is real HTML in the document, so it is in the first paint and in a
crawler's view of the page. Everything inside the shadow root appears once the
module loads. `alacris.Pending` is the stylesheet that keeps that transition
from flashing.

**Every prop crosses as an attribute, objects and arrays included.** alacris
coerces an attribute using the type of the prop's default, and `JSON.parse` is
what it uses for object and array defaults. A fully-formed component needs no
post-load property assignment, and the page is complete before any JavaScript
has run.

## Layers

Each works on its own.

| | |
| --- | --- |
| `alacris` | render elements, serve the runtime. Stateless, ordinary request/response. |
| `gen` + `alacris-go` | generate typed Go wrappers from your `define()` calls. |
| `live` | push prop changes from the server, receive component events. |
| `app` | the same live handler, in an OS webview. Nested module, `-tags desktop`. |

### 1. Rendering

`Element` implements `templ.Component`, so it drops straight into a `.templ`
file and picks up children from the enclosing block:

```templ
@alacris.E("user-card").Prop("name", "Ada").Prop("tags", []string{"math", "code"}) {
    <h3 slot="title">Ada Lovelace</h3>
}
```

Serve the runtime and emit the script tags:

```go
mux.Handle("/_alacris/", alacris.RuntimeHandler())
```

```templ
<head>
    @ui.Pending()
    @alacris.Scripts(alacris.Config{
        UI:      true,                 // Material Design 3 catalog + theme
        Modules: []string{"/web/components.js"},
        Version: build.Revision,   // makes each release a distinct URL
    })
</head>
```

`Scripts` writes the import map (`alacris`, `alacris/store`, `alacris/context`,
`alacris/signal`) and your module entry points. Set `Config.UI` to also load
Alacris UI (`@alacris/core` points at the same bytes as `alacris`, so the page
has one reactive graph). It belongs in `<head>`: an import map has to precede
the first module import it applies to.

### 2. Generating typed wrappers

Write components in JavaScript, because `setup()` runs in the browser:

```js
/**
 * A person, at a glance.
 *
 * @prop {string[]} tags   the labels shown under the name
 * @fires greet {name: string} - the user said hello
 * @slot  title - replaces the heading
 * @cssprop [--card-bg=#fff] - the background
 */
define('user-card', {
  props: { name: 'anon', age: 0, tags: [] },
  setup({ name, age, tags }, host) { /* ... */ },
});
```

```bash
alacris-go generate ./web/components -o ./internal/components
```

You get a `UserCardProps` struct, a `UserCard` function returning an
`*alacris.Element`, typed event details, slot name constants, and the theming
contract:

```go
@components.UserCard(components.UserCardProps{Name: "Ada", Age: 36}).
    Apply(components.UserCardVars, map[string]string{"--card-bg": "#ffe9a8"}) {
    <span slot={ components.UserCardSlotTitle }>Ada Lovelace</span>
}
```

Because the generator knows each prop's default, a value equal to it is left
off the element. Smaller HTML, and only this layer can know to do it.

The scanner reads the `define()` call itself, so there is one source of truth
and no second file to keep in step. It is a scanner, not a JavaScript engine: a
props object it cannot read as literal data is **an error, never a guess**. When
a component builds its props at runtime, write it into a manifest by hand and
generate from that. `generate` accepts either.

```
alacris-go generate <path>... -o <dir>   write Go wrappers
alacris-go check    <path>... -o <dir>   fail if they are out of date (for CI)
alacris-go manifest <path>...            write what it found, as JSON
```

Wire it next to the templ step:

```go
//go:generate go run github.com/bmartel/alacris-go/cmd/alacris-go generate ./web -o ./internal/components -strip ala-
//go:generate go run github.com/a-h/templ/cmd/templ@latest generate
```

#### JSDoc tags

`define()` says what a prop is called and how it is coerced. These say the rest.

| Tag | What it does |
| --- | --- |
| `@prop {type} name  description` | overrides the inferred Go type, documents the field |
| `@fires name {a: string} - description` | generates an event constant and a typed detail struct |
| `@slot name - description` | generates a slot name constant |
| `@cssprop [--x=default] - description` | generates the theming contract |
| `@goname Name` | overrides the derived Go identifier |
| `@goimport alias path` | imports a package for a `go:` type |

Types: `string`, `number` (float64), `integer`, `boolean`, `object`, `any`,
`T[]`, `Array<T>`, `Record<string, T>`, and `go:YourType` for a Go type you
declared by hand.

### 3. Server-driven reactivity

```go
srv := live.New()
defer srv.Close()

mux := http.NewServeMux()
live.Mount(mux, alacris.DefaultBase, srv)   // runtime + client + endpoints
```

Per page render, mint a session and put it in the page:

```go
// Reads or sets the cookie that authorises this browser, so call it before
// writing anything to w.
sess := srv.NewSession(w, r)
sess.OnOpen(func(s *live.Session) { push(s) })   // also runs after a reconnect

cfg := alacris.Config{Live: true, Page: sess.ID(), /* ... */}
```

Patches, one property write per change, coalesced into one frame:

```go
sess.Batch(func() {
    sess.Element("board").Set("items", list.Items())
})
```

Actions: a component's `CustomEvent`s forwarded to named server actions:

```go
@ui.Board(props).ID("board").On(ui.BoardEventAdd, "add-card")
```

```go
live.On(srv, "add-card", func(c *live.Ctx, d ui.BoardAddDetail) error { ... })
```

One delegated listener per event type covers every element, present and future;
alacris events are composed and bubbling, so it works across shadow boundaries.

The transport is SSE down and an ordinary POST up. No WebSocket, no extra
dependency. `Handle.SetHTML` is there for the cases props express badly, but
props are the better tool nearly every time.

This layer costs a stateful server and session affinity behind a load balancer.
The first two layers do not depend on it.

## Prop encoding

The Go type decides the encoding, and it has to agree with the type of the
prop's default in `define()`. Generated wrappers guarantee that.

| Go | attribute | matching default |
| --- | --- | --- |
| `string`, `fmt.Stringer`, `time.Time` | text | a string |
| `bool` | `"true"` / `"false"` | a boolean |
| integers, floats | a number | a number |
| slices, maps, structs | JSON | an object or array |

Three rules exist because of sharp edges in the runtime:

- **Booleans are always written out**, never signalled by presence. Removing an
  attribute runs `coerce(null, default)`, which returns `false` even when the
  declared default is `true`.
- **A boolean prop whose default is `true` generates a `*bool` field.** Go's
  zero value is `false`, so a plain field could never mean "leave it alone" and
  the component's default would be unreachable.
- **Integers past 2<sup>53</sup>−1 are an error, not a rounding.** The value is
  coerced with `+v` on the other side. Send large identifiers as strings.

A prop patch sent by `live` uses the **JavaScript** prop name (`maxCount`),
because it writes the DOM property. A server-rendered prop uses the same name
and this library kebab-cases it into the attribute (`max-count`) exactly the way
`define.js` does, quirks included.

## Security

- **The live capability is an `HttpOnly` cookie**, set by `NewSession` with
  `SameSite=Lax`, a path scoped to the live endpoints, and `Secure` over TLS.
  It never appears in a page, a URL or a log, and script cannot read it. What
  the page carries is a page id, which is an identifier and not a secret:
  without the cookie it reaches nothing. Serve pages that set the cookie
  `no-store`.
- **Action payloads are input.** A well-behaved component emits what it says it
  emits; a console can emit anything. Binding is strict, the body is size-capped,
  and cross-origin posts are refused. Validate what you decode.
- Interpolated prop values are attribute values, never markup. There is nothing
  to escape and no way to forget.
- `Style` and `Var` refuse anything that could escape a declaration, rather than
  silently substituting a placeholder.
- Under a Trusted Types CSP, allow both policies:
  `Content-Security-Policy: trusted-types alacris alacris-live;`
  `Config.Nonce` (or `templ.WithNonce`) puts a nonce on every script tag.

## The example

```bash
go run ./examples/todo
# http://localhost:8080

go run -tags desktop ./examples/todo -desktop
```

A live board the server owns. Components in JavaScript, wrappers generated
from them, card state in Go, every change arriving as one prop write. Open it
in two windows. Move a card in one and it slides columns in the other;
whatever you were typing does not. Neither tab polls.

```bash
go run ./examples/todo -demo
```

`-demo` is a collaborator that moves cards on a timer, so a one-window
recording is enough to film the same trick.

Each lane has its own `each()`, so the columns are real stacks. A card that
stays in a lane keeps its node when the list is rewritten; a card that changes
lane is created in the destination. That is the cost of the layout, and the
identity tests reorder inside a lane so they still catch an `each` placed
inside a conditional.

## Vendored runtime

`assets/` holds alacris `RuntimeVersion`, published to npm, byte-pinned by
`assets_test.go`. Refresh it with:

```bash
go run ./internal/vendorjs             # fetch and write
go run ./internal/vendorjs -check      # verify, change nothing
```

A vendored copy of someone else's build goes stale silently. The failing test
is how you find out.

## Building with AI agents

[`AGENTS.md`](https://bmartel.github.io/alacris-go/AGENTS.md) is a drop-in file
that teaches coding agents the conventions here: the encoding rules, that `ui/`
is generated, that `each` must not sit inside a conditional, that a reconnecting
page needs `OnOpen`. Put it in your project root:

```bash
curl -o AGENTS.md https://bmartel.github.io/alacris-go/AGENTS.md
```

There is also an [`llms.txt`](https://bmartel.github.io/alacris-go/llms.txt) map
of the documentation for agents that fetch docs on demand.

## Documentation

Full documentation, with every Go example rendered by the library itself, is at
**[bmartel.github.io/alacris-go](https://bmartel.github.io/alacris-go/)**.

Every example on the site is extracted from `internal/docsgen/examples.go` with
`go/ast` and then executed. The HTML shown beside the Go is what that Go
actually rendered, and `go test ./...` fails if the two stop matching.

## Development

```bash
go test ./...
cd app && go test ./...
go test -race ./...
go generate ./examples/todo/...

go run ./internal/docsgen        # re-render the documentation examples
cd docs && npm install && npm run dev
```

Browser tests cover the claims `go test` cannot reach: that a server-driven
update moves rows instead of rebuilding them, that focus and a half-typed draft
survive it, and that every open tab stays in step:

```bash
cd e2e && npm install && npx playwright install chromium
npx playwright test
```

They run against `examples/todo` unmodified.

## License

MIT. The vendored alacris runtime is MIT too; see `assets/LICENSE.alacris`.
