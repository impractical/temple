package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type LinkedCSSSite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type LinkedCSSHomePage struct {
	Layout LinkedCSSLayout
	User   string
}

func (LinkedCSSHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h LinkedCSSHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (LinkedCSSHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h LinkedCSSHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (LinkedCSSHomePage) LinkCSS(_ context.Context) []temple.CSSLink {
	return []temple.CSSLink{
		{Href: "https://example.com/a.css", Rel: "stylesheet"},
	}
}

type LinkedCSSLayout struct {
}

func (b LinkedCSSLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (LinkedCSSLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_linkedCSS() {
	// normally you'd use something like embed.FS or os.DirFS for this
	// for example purposes, we're just hardcoding values
	var templates = staticFS{
		"home.html.tmpl": `{{ define "body" }}Hello, {{ .Page.User }}. This is my home page.{{ end }}`,
		"base.html.tmpl": `
<!doctype html>
<html lang="en">
	<head>
		<title>{{ .Site.Title }}</title>
		{{- .CSS -}}
		{{- .HeaderJS -}}
	</head>
	<body>
		{{ block "body" . }}{{ end }}
		{{- .FooterJS -}}
	</body>
</html>`,
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := LinkedCSSSite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := LinkedCSSHomePage{
		Layout: LinkedCSSLayout{},
		User:   "Visitor",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><link href="https://example.com/a.css" rel="stylesheet">
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.</body>
	// </html>
}
