package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type MySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type HomePage struct {
	Layout BaseLayout
}

func (HomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h HomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (HomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h HomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (HomePage) EmbedCSS(_ context.Context) []temple.CSSInline {
	return []temple.CSSInline{
		{TemplatePath: "home.css.tmpl"},
	}
}

type BaseLayout struct {
}

func (b BaseLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (BaseLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_basic() {
	// normally you'd use something like embed.FS or os.DirFS for this
	// for example purposes, we're just hardcoding values
	var templates = staticFS{
		"home.html.tmpl": `{{ define "body" }}Hello, world. This is my home page.{{ end }}`,
		"base.html.tmpl": `
<!doctype html>
<html lang="en">
	<head>
		<title>{{ .Site.Title }}</title>
		{{ .CSS }}
	</head>
	<body>
		{{ block "body" . }}{{ end }}
	</body>
</html>`,
		"home.css.tmpl": "body { font: red; }",
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := MySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := HomePage{Layout: BaseLayout{}}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title>
	// 		<style>
	// body { font: red; }
	// </style>
	//
	// 	</head>
	// 	<body>
	// 		Hello, world. This is my home page.
	// 	</body>
	// </html>
}
