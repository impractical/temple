package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type LinkedJSSite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type LinkedJSHomePage struct {
	Layout LinkedJSLayout
	User   string
}

func (LinkedJSHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h LinkedJSHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (LinkedJSHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h LinkedJSHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (LinkedJSHomePage) LinkJS(_ context.Context) []temple.JSLink {
	return []temple.JSLink{
		{Src: "https://example.com/a.js"},
		{Src: "https://example.com/b.js", PlaceInFooter: true},
	}
}

type LinkedJSLayout struct {
}

func (b LinkedJSLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (LinkedJSLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_linkedJS() {
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

	site := LinkedJSSite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := LinkedJSHomePage{
		Layout: LinkedJSLayout{},
		User:   "Visitor",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><script src="https://example.com/a.js"></script>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.<script src="https://example.com/b.js"></script>
	// </body>
	// </html>
}
