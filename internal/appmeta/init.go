package appmeta

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// InitOptions control what `alacris-go app init` writes.
type InitOptions struct {
	Dir        string
	Name       string
	Identifier string
	Module     string
}

// Init writes a desktop-first alacris-go project into dir.
func Init(opts InitOptions) error {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	if opts.Name == "" {
		base := filepath.Base(opts.Dir)
		if base == "." || base == string(filepath.Separator) {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			base = filepath.Base(wd)
		}
		opts.Name = base
	}
	if opts.Identifier == "" {
		opts.Identifier = "com.example." + slug(opts.Name)
	}
	if opts.Module == "" {
		opts.Module = "example.com/" + slug(opts.Name)
	}
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return err
	}
	data := scaffoldData{
		Name:       opts.Name,
		Identifier: opts.Identifier,
		Module:     opts.Module,
		Slug:       slug(opts.Name),
	}
	files := map[string]string{
		FileName:            appJSON,
		"go.mod":            goMod,
		"main.go":           mainGo,
		"page.templ":        pageTempl,
		"web/components.js": componentsJS,
	}
	for name, src := range files {
		tmpl, err := template.New(name).Parse(src)
		if err != nil {
			return err
		}
		path := filepath.Join(opts.Dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("appmeta: %s already exists", path)
		}
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		err = tmpl.Execute(f, data)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

type scaffoldData struct {
	Name, Identifier, Module, Slug string
}

func (s scaffoldData) Package() string {
	p := slug(s.Name)
	p = strings.ReplaceAll(p, "-", "")
	if p == "" {
		return "main"
	}
	return p
}

const appJSON = `{
  "name": "{{.Name}}",
  "identifier": "{{.Identifier}}",
  "version": "0.1.0",
  "main": "."
}
`

const goMod = `module {{.Module}}

go 1.26.5

require (
	github.com/a-h/templ v0.3.1020
	github.com/bmartel/alacris-go v0.0.0
	github.com/bmartel/alacris-go/app v0.0.0
)
`

const mainGo = `package main

import (
	"log"
	"log/slog"
	"net/http"

	alacris "github.com/bmartel/alacris-go"
	"github.com/bmartel/alacris-go/app"
	"github.com/bmartel/alacris-go/live"
)

//go:generate go run github.com/bmartel/alacris-go/cmd/alacris-go generate ./web -o ./internal/components
//go:generate go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate

func main() {
	mux := http.NewServeMux()
	srv := live.New(live.Options{
		Logger:       slog.Default(),
		CookieSecure: live.SecureNever,
	})
	defer srv.Close()

	live.Mount(mux, alacris.DefaultBase, srv)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		sess := srv.NewSession(w, r)
		sess.OnOpen(func(s *live.Session) {})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, private")
		cfg := alacris.Config{UI: true, Live: true, Page: sess.ID(), Modules: []string{"/web/components.js"}}
		if err := page(cfg).Render(r.Context(), w); err != nil {
			slog.Error("render", "error", err)
		}
	})
	mux.Handle("/web/", http.FileServer(http.Dir(".")))

	if err := app.Run(app.Options{
		Title:   "{{.Name}}",
		Width:   1024,
		Height:  768,
		Handler: mux,
		Menu:    app.DefaultMenu(),
	}); err != nil {
		log.Fatal(err)
	}
}
`

const pageTempl = `package main

import (
	alacris "github.com/bmartel/alacris-go"
	"github.com/bmartel/alacris-go/ui"
)

templ page(cfg alacris.Config) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{{.Name}}</title>
			@ui.Pending()
			@alacris.Scripts(cfg)
		</head>
		<body>
			@ui.AppBar(ui.AppBarProps{}) {
				{{.Name}}
			}
			@ui.Container(ui.ContainerProps{}) {
				@ui.Text(ui.TextProps{}) {
					The live server and this window are the same process.
					Call app.SaveFile from a live.On handler; do not invent a JS bridge.
				}
			}
		</body>
	</html>
}
`

const componentsJS = `import { define, html } from 'alacris';

define('hello-world', {
  props: { name: 'world' },
  setup: ({ name }) => html` + "`" + `<p>Hello, ${name}!</p>` + "`" + `,
});
`
