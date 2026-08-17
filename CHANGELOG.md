# Changelog

All notable changes to this project are documented here. Releases are cut
automatically by [semantic-release](https://semantic-release.gitbook.io/) from
[Conventional Commits](https://www.conventionalcommits.org/).

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
