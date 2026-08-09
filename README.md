<div align="center">

# alacris-go

**Build [alacris](https://github.com/bmartel/alacris) web components from Go and [templ](https://templ.guide).**

Typed wrappers generated from your `define()` calls · the runtime served from Go, no npm · server-driven props with no HTML on the wire

</div>

```go
@ui.TodoList(ui.TodoListProps{Items: todos, Filter: "active"}).
    ID("todos").
    On(ui.TodoListEventAdd, "add-todo")
```

```go
live.On(srv, "add-todo", func(c *live.Ctx, d ui.TodoListAddDetail) error {
    list.Add(d.Text)
    c.Session.Element("todos").Set("items", list.Items())   // one property write
    return nil
})
```

That second block is the whole idea. An alacris prop is a signal, so "the server
changed something" compiles to a single DOM property write and a single node
update — no HTML over the wire, nothing to diff, nothing to morph, and nothing
that disturbs focus, scroll position or what the user has typed.

## Install

```bash
go get github.com/bmartel/alacris-go
```

The alacris runtime is vendored into the module and served from Go, so a Go
project needs no npm at all.

## What the server can and cannot render

Worth being clear about up front, because it shapes everything else.

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
module loads — `alacris.Pending` is the stylesheet that keeps that transition
from flashing.

The useful consequence: **every prop crosses as an attribute, objects and arrays
included.** alacris coerces an attribute using the type of the prop's default,
and `JSON.parse` is what it uses for object and array defaults. A fully-formed
component needs no post-load property assignment, and the page is complete
before any JavaScript has run.

## Three layers

Each works on its own.

| | |
| --- | --- |
| `alacris` | render elements, serve the runtime. Stateless, ordinary request/response. |
| `gen` + `alacris-go` | generate typed Go wrappers from your `define()` calls. |
| `live` | push prop changes from the server, receive component events. |

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
    @alacris.Pending{Tags: ui.Tags}
    @alacris.Scripts(alacris.Config{
        Modules: []string{"/web/components.js"},
        Version: build.Revision,   // makes each release a distinct URL
    })
</head>
```

`Scripts` writes the import map (`alacris`, `alacris/store`, `alacris/context`,
`alacris/signal`) and your module entry points. It belongs in `<head>`: an
import map has to precede the first module import it applies to.

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
alacris-go generate ./web/components -o ./internal/ui
```

You get a `UserCardProps` struct, a `UserCard` function returning an
`*alacris.Element`, typed event details, slot name constants, and the theming
contract:

```go
@ui.UserCard(ui.UserCardProps{Name: "Ada", Age: 36}).
    Apply(ui.UserCardVars, map[string]string{"--card-bg": "#ffe9a8"}) {
    <span slot={ ui.UserCardSlotTitle }>Ada Lovelace</span>
}
```

Because the generator knows each prop's default, a value equal to it is left off
the element — smaller HTML, and only this layer can know to do it.

The scanner reads the `define()` call itself, so there is one source of truth
and no second file to keep in step. It is a scanner, not a JavaScript engine: a
props object it cannot read as literal data is **an error, never a guess**. When
a component builds its props at runtime, write it into a manifest by hand and
generate from that — `generate` accepts either.

```
alacris-go generate <path>... -o <dir>   write Go wrappers
alacris-go check    <path>... -o <dir>   fail if they are out of date (for CI)
alacris-go manifest <path>...            write what it found, as JSON
```

Wire it next to the templ step:

```go
//go:generate go run github.com/bmartel/alacris-go/cmd/alacris-go generate ./web -o ./ui -strip ala-
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
sess := srv.NewSession()
sess.OnOpen(func(s *live.Session) { push(s) })   // also runs after a reconnect

cfg := alacris.Config{Live: true, Session: sess.ID(), /* ... */}
```

Down — one property write per change, coalesced into one frame:

```go
sess.Batch(func() {
    sess.Element("todos").Set("items", list.Items())
    sess.Element("todos").Set("filter", "active")
})
```

Up — a component's `CustomEvent`s forwarded to named server actions:

```go
@ui.TodoList(props).ID("todos").On(ui.TodoListEventAdd, "add-todo")
```

```go
live.On(srv, "add-todo", func(c *live.Ctx, d ui.TodoListAddDetail) error { ... })
```

One delegated listener per event type covers every element, present and future;
alacris events are composed and bubbling, so it works across shadow boundaries.

The transport is SSE down and an ordinary POST up — no WebSocket, no extra
dependency. `Handle.SetHTML` is there for the cases props express badly, but
props are the better tool nearly every time.

This layer costs a stateful server and session affinity behind a load balancer.
The first two layers do not depend on it.

## Prop encoding

The Go type decides the encoding, and it has to agree with the type of the
prop's default in `define()` — which generated wrappers guarantee.

| Go | attribute | matching default |
| --- | --- | --- |
| `string`, `fmt.Stringer`, `time.Time` | text | a string |
| `bool` | `"true"` / `"false"` | a boolean |
| integers, floats | a number | a number |
| slices, maps, structs | JSON | an object or array |

Three rules exist because of sharp edges in the runtime, and they are the reason
this is a library rather than a snippet:

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

- **A session id is a capability.** It is generated with `crypto/rand`, and
  anyone holding it can receive that page's patches and act as that page. It
  belongs in the page it was made for and nowhere else — not in a URL that might
  be shared, logged or sent as a referer. Serve those pages `no-store`.
- **Action payloads are input.** A well-behaved component emits what it says it
  emits; a console can emit anything. Binding is strict, the body is size-capped,
  and cross-origin posts are refused. Validate what you decode.
- Interpolated prop values are attribute values, never markup — there is nothing
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
```

A todo list the server owns. Components in JavaScript, wrappers generated from
them, list state in Go, every change arriving as one prop write. Open it in two
tabs — they stay in step, and neither polls.

It is also where the non-obvious parts are demonstrated honestly: the list is
**not** wrapped in a conditional template, because that would rebuild every row
on every update and undo the thing `each` exists to do.

## Vendored runtime

`assets/` holds alacris `RuntimeVersion`, published to npm, byte-pinned by
`assets_test.go`. Refresh it with:

```bash
go run ./internal/vendorjs             # fetch and write
go run ./internal/vendorjs -check      # verify, change nothing
```

A vendored copy of someone else's build goes stale silently; the failing test is
the point.

## Development

```bash
go test ./...
go test -race ./...
go generate ./examples/todo/...
```

## License

MIT. The vendored alacris runtime is MIT too — see `assets/LICENSE.alacris`.
