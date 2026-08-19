# Changelog

All notable changes to this project are documented here. Releases are cut
automatically by [semantic-release](https://semantic-release.gitbook.io/) from
[Conventional Commits](https://www.conventionalcommits.org/).

## [0.9.0](https://github.com/bmartel/alacris-go/compare/v0.8.3...v0.9.0) (2026-08-19)

### Features

* vendor alacris 0.10.0 and @alacris/ui 0.2.3 ([e7b5ff7](https://github.com/bmartel/alacris-go/commit/e7b5ff78149c10405c7e3da0fe89186aca0b42d7))

## [0.8.3](https://github.com/bmartel/alacris-go/compare/v0.8.2...v0.8.3) (2026-08-19)

### Bug Fixes

* an app icon is a ladder of sizes, and each rung says which it is ([70a4989](https://github.com/bmartel/alacris-go/commit/70a4989b5d4d977c7d5fba7d9aa1afd4951bcb86))

## [0.8.2](https://github.com/bmartel/alacris-go/compare/v0.8.1...v0.8.2) (2026-08-19)

### Bug Fixes

* Escape in a popup no longer closes the dialog around it ([73b1477](https://github.com/bmartel/alacris-go/commit/73b147716de8027774dfddd6406769cccbe87267))
* regenerate the UI wrappers for 0.2.2 ([97b9c6d](https://github.com/bmartel/alacris-go/commit/97b9c6d9cea3eac6a02badb5d8352edcb5cbc202))

## [0.8.1](https://github.com/bmartel/alacris-go/compare/v0.8.0...v0.8.1) (2026-08-19)

### Bug Fixes

* **app:** an inset title bar you can drag by ([7281db8](https://github.com/bmartel/alacris-go/commit/7281db8764a081d946903789a76fe6a4d1fdf500))
* **app:** double clicking an inset title bar zooms the window ([c257c1b](https://github.com/bmartel/alacris-go/commit/c257c1bf75e48767c2e4f9413040f835ff03e3dd))

## [0.8.0](https://github.com/bmartel/alacris-go/compare/v0.7.3...v0.8.0) (2026-08-19)

### Features

* **app:** a title bar the page can draw behind ([92faaa2](https://github.com/bmartel/alacris-go/commit/92faaa2804539ca51f427cc60f8562de502ce75a))

## [0.7.3](https://github.com/bmartel/alacris-go/compare/v0.7.2...v0.7.3) (2026-08-18)

### Bug Fixes

* make the generator and nested app module usable from a consumer app ([74ab9b9](https://github.com/bmartel/alacris-go/commit/74ab9b9ea95ca97a595bed9462ad899fb75cc83a)), closes [#2](https://github.com/bmartel/alacris-go/issues/2) [#3](https://github.com/bmartel/alacris-go/issues/3) [#4](https://github.com/bmartel/alacris-go/issues/4)

## [0.7.2](https://github.com/bmartel/alacris-go/compare/v0.7.1...v0.7.2) (2026-08-18)

### Bug Fixes

* authenticate the single-instance channel ([121db1f](https://github.com/bmartel/alacris-go/commit/121db1fa29e3cd7f8f10846fc25c304ca33d80ac))
* bound pointer indirection in EncodeProp ([6442d93](https://github.com/bmartel/alacris-go/commit/6442d9348b75733694a8e7b3aad7d78ca4240950))
* deliver live patches under the session lock ([2dce0fe](https://github.com/bmartel/alacris-go/commit/2dce0fe44ea41dc5509c8ab583d2677f464c991e))
* keep the Windows instance lock off the handover bytes ([306bdb3](https://github.com/bmartel/alacris-go/commit/306bdb3777cf01def7b3fa9071ea41fbc7fe8393))
* re-prefix every line of a generated doc comment ([c1470a8](https://github.com/bmartel/alacris-go/commit/c1470a8e46b22f69a1b691620a9b3244741e703e))
* reject control characters in app bundle metadata ([68ba772](https://github.com/bmartel/alacris-go/commit/68ba772cf098146f4ff41cf5fe12db672712a3ae))
* restrict OpenURL to safe schemes and require https for updates ([974b714](https://github.com/bmartel/alacris-go/commit/974b71470da5f7b629c4b840b096f47604e285ba))

### Performance

* batch live session eviction at the cap ([fd1ecae](https://github.com/bmartel/alacris-go/commit/fd1ecae762f55264283838007c2756cc4932ebd4))

## [0.7.1](https://github.com/bmartel/alacris-go/compare/v0.7.0...v0.7.1) (2026-08-17)

### Bug Fixes

* compile the desktop host on Linux and Windows ([efa76ed](https://github.com/bmartel/alacris-go/commit/efa76ed5ec2145ed30d0b8a00dceb62f659a7286))

## [0.7.0](https://github.com/bmartel/alacris-go/compare/v0.6.0...v0.7.0) (2026-08-17)

### Features

* close the desktop host against a Tauri-shaped product ([53531e7](https://github.com/bmartel/alacris-go/commit/53531e7cd917d27d2ee14875c3b4fefe3f480f98))

## [0.6.0](https://github.com/bmartel/alacris-go/compare/v0.5.0...v0.6.0) (2026-08-17)

### Features

* host the live app in a native OS webview ([8d499ca](https://github.com/bmartel/alacris-go/commit/8d499cae59529617ca1985aa684feca149be50b9))

## [0.5.0](https://github.com/bmartel/alacris-go/compare/v0.4.0...v0.5.0) (2026-08-17)

### Features

* **examples:** turn the todo fixture into a live Kanban board ([2e7542b](https://github.com/bmartel/alacris-go/commit/2e7542b2f95e785fe2cdb7dbd6ec01ee0dd6adfc))

## [0.4.0](https://github.com/bmartel/alacris-go/compare/v0.3.0...v0.4.0) (2026-08-16)

### Features

* ship Alacris UI as typed Material Design 3 components ([de98acf](https://github.com/bmartel/alacris-go/commit/de98acf3857d3aa6123ba8dd73ada70f6822ff55))

## [0.3.0](https://github.com/bmartel/alacris-go/compare/v0.2.0...v0.3.0) (2026-08-14)

### Features

* **gen:** emit typed live patch handles ([2f641a6](https://github.com/bmartel/alacris-go/commit/2f641a6e3cf50e51735b0f9b301000cd66089cb8))
* **live:** hardening, hot-path performance, and dead-session recovery ([3ae429e](https://github.com/bmartel/alacris-go/commit/3ae429eaf892e6106c3227cc6773ec8c9c7ab772))
* **live:** livetest, so action handlers are unit-testable ([f16a88c](https://github.com/bmartel/alacris-go/commit/f16a88c5795a9cf607d09314bc815c66de9ecf8d))
* **vendor:** vendor alacris 0.3.0 and automate staying current ([8a22d1f](https://github.com/bmartel/alacris-go/commit/8a22d1f7644bb6973aad9a18a2ca38fd9386823a))

### Bug Fixes

* **gen:** refuse manifest strings that are more than what they claim to be ([ced05f3](https://github.com/bmartel/alacris-go/commit/ced05f3cbd00092c365e205fdce83e1e3b61517f))

## [0.2.0](https://github.com/bmartel/alacris-go/compare/v0.1.2...v0.2.0) (2026-08-09)

### ⚠ BREAKING CHANGES

* **live:** Server.NewSession now takes the http.ResponseWriter and
*http.Request, because it reads or sets the cookie and must run before anything
is written. alacris.Config.Session is renamed to Config.Page, since the value
it carries is an identifier rather than a capability. The wire protocol changed
with it: ?s= is ?p=, the action body key s is p, the script attribute
data-session is data-page, and the action endpoint requires an
application/json content type. All three are compile errors or a 404; nothing
changes behaviour silently. See docs/start/upgrading for the migration.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>

### Features

* **live:** authorise the stream with a cookie, not a URL parameter ([2602170](https://github.com/bmartel/alacris-go/commit/26021703155360f4341aa42c8d19c205957d93f4))

## [0.1.2](https://github.com/bmartel/alacris-go/compare/v0.1.1...v0.1.2) (2026-08-09)

### Bug Fixes

* **alacris:** format named float32 props at 32 bits ([4d76a62](https://github.com/bmartel/alacris-go/commit/4d76a62ce0517dcebcc95cf5c012dcf11dc162a5))
* **alacris:** validate prop and slot-wrapper names before they reach the tag ([3198b58](https://github.com/bmartel/alacris-go/commit/3198b582cd33f61a8dbff99a0d496bb9cb125078))
* **gen:** reduce a source path to a base name inside fileBase ([2ebda66](https://github.com/bmartel/alacris-go/commit/2ebda662b1ef44baba1e88596b651bfd01c53735))
* **live:** resync a slow subscriber and bound how many sessions are held ([c8926f6](https://github.com/bmartel/alacris-go/commit/c8926f6e8e0da90a942f5b2fee464e42da7bd836))

## [0.1.1](https://github.com/bmartel/alacris-go/compare/v0.1.0...v0.1.1) (2026-08-09)

### Bug Fixes

* **live:** give sessions their own context, outliving the request ([26b8dc7](https://github.com/bmartel/alacris-go/commit/26b8dc782a6050653a7af3ba0eb61976b46574f4))

## 0.1.0 (2026-08-09)

The first release. Three layers, each usable on its own.

### Features

* **alacris:** an `Element` that implements `templ.Component`, with prop
  encoding that matches alacris' coercion rules — including the two that exist
  because of sharp edges in the runtime: booleans are always written out rather
  than signalled by presence, and integers past JavaScript's safe range are an
  error rather than a silent rounding.
* **alacris:** the alacris runtime vendored into the module and served by
  `RuntimeHandler`, so a Go project needs no npm. `internal/vendorjs` refreshes
  it and `assets_test.go` pins the bytes.
* **alacris:** `Scripts` for the import map and module tags, `Pending` for the
  flash before elements are defined, and `VarSet` for a component's theming
  contract.
* **gen:** a JavaScript scanner that reads the props object out of `define()`
  calls and emits typed Go wrappers, typed event details, slot constants and
  theming contracts. A props object it cannot read as literal data is an error,
  never a guess; a hand-editable manifest is the escape hatch.
* **cmd/alacris-go:** `generate`, `check` and `manifest`. Output does not depend
  on the working directory, so `check` behaves the same locally and in CI.
* **live:** server-driven props over SSE and component events over POST. Because
  a prop is a signal, a change is one property write, one binding and one DOM
  node — no HTML on the wire, and focus, scroll position and typed text survive.
  Session ids are `crypto/rand` capabilities.

### Documentation

* A full documentation site at
  [bmartel.github.io/alacris-go](https://bmartel.github.io/alacris-go/), where
  every Go example is extracted from `internal/docsgen/examples.go` and executed,
  so an example cannot drift from the library.
* `AGENTS.md`, a drop-in conventions file for coding agents, and `llms.txt`.
