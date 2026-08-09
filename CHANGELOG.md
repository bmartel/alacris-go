# Changelog

All notable changes to this project are documented here. Releases are cut
automatically by [semantic-release](https://semantic-release.gitbook.io/) from
[Conventional Commits](https://www.conventionalcommits.org/).

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
