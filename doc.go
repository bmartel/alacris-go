// Package alacris renders alacris web components from Go and templ.
//
// alacris (https://github.com/bmartel/alacris) is a ~6 kB web component
// library built on signals and fine-grained DOM updates. This package is the
// server half: it renders the tags, gets the props across the boundary
// correctly, and serves the runtime.
//
// # What the server can and cannot render
//
// A component's shadow content is produced by its setup() function when the
// element connects, in the browser. There is no server-side rendering of
// component internals and no declarative shadow DOM support. What Go renders
// is the element itself:
//
//	<user-card name="Ada" age="36">
//	  <h3 slot="title">Ada Lovelace</h3>   <!-- light DOM, visible immediately -->
//	</user-card>
//
// Slot content is real HTML in the document, so it is in the first paint and
// in the crawler's view of the page. Everything inside the shadow root appears
// once the module loads — see Pending for the stylesheet that keeps that
// transition from flashing.
//
// # Props
//
// Every prop crosses as an attribute, including objects and arrays: alacris
// coerces an attribute using the type of the prop's default, and JSON.parse is
// what it uses for object and array defaults. That means a fully-formed
// component needs no post-load property assignment, and the page works with
// JavaScript still in flight.
//
// The encoding rules are in EncodeProp. Two of them exist because of sharp
// edges in the runtime:
//
//   - Booleans are always written out as "true" or "false" rather than by
//     presence. Removing an attribute runs coerce(null, default), which
//     returns false even when the declared default is true.
//   - Integers beyond JavaScript's safe range are an error, not a rounding.
//     Send large identifiers as strings.
//
// # Layers
//
// The package divides into three, each usable on its own:
//
//	alacris  — render elements and serve the runtime. Stateless.
//	gen      — generate typed Go wrappers from your define() calls.
//	live     — push prop changes from the server and receive component events.
//
// The first two are ordinary request/response rendering. The third makes the
// server authoritative over component state: because a prop is a signal,
// "the server changed something" compiles to a single property write and a
// single DOM node update, with no HTML on the wire and no morphing.
//
// Alacris UI ships in the ui subpackage: sixty-eight Material Design 3
// components, enabled with Config.UI. A zero Config.Theme still applies the
// Material default scheme.
//
// # Content Security Policy
//
// alacris registers a Trusted Types policy named by TrustedTypesPolicy for
// template parsing, and contains no eval. The live client registers a second,
// named alacris-live, which it uses only for Handle.SetHTML. Under a
// trusted-types directive, allow the ones you use:
//
//	Content-Security-Policy: trusted-types alacris alacris-live; require-trusted-types-for 'script'
//
// Config.Nonce (or templ.WithNonce on the context) puts a nonce on every
// script tag this package emits.
package alacris

//go:generate go run ./internal/vendorjs
//go:generate go run ./internal/genui
