module github.com/bmartel/alacris-go

go 1.26.5

require github.com/a-h/templ v0.3.1020

// Nested module. A replace in this file is ignored by consumers, so the
// version here has to exist on the module proxy: a pseudo-version of a
// pushed commit, or app/vX.Y.Z once a release has tagged the nested module
// in lockstep. Never v0.0.0.
require github.com/bmartel/alacris-go/app v0.0.0-20260818005401-8b871f97a92b

// Indirect via app. Linked only under -tags desktop; the root packages do not import it.
require github.com/webview/webview_go v0.0.0-20240831120633-6173450d4dd6 // indirect

replace github.com/bmartel/alacris-go/app => ./app
