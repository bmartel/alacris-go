module github.com/bmartel/alacris-go

go 1.26.5

require github.com/a-h/templ v0.3.1020

require github.com/bmartel/alacris-go/app v0.0.0

// Indirect via app. Linked only under -tags desktop; the root packages do not import it.
require github.com/webview/webview_go v0.0.0-20240831120633-6173450d4dd6 // indirect

replace github.com/bmartel/alacris-go/app => ./app
