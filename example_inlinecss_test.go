package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type InlineCSSSite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type InlineCSSHomePage struct {
	Layout    InlineCSSLayout
	User      string
	FontColor string
}

func (InlineCSSHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h InlineCSSHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (InlineCSSHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h InlineCSSHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (InlineCSSHomePage) EmbedCSS(_ context.Context) []temple.CSSInline {
	return []temple.CSSInline{
		{TemplatePath: "home.css.tmpl"},
	}
}

type InlineCSSLayout struct {
}

func (b InlineCSSLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (InlineCSSLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_inlineCSS() {
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
		"home.css.tmpl": "body { font: {{ .Page.FontColor }}; }",
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := InlineCSSSite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := InlineCSSHomePage{
		Layout:    InlineCSSLayout{},
		User:      "Visitor",
		FontColor: "red",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><style>
	// body { font: red; }
	// </style>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.</body>
	// </html>
}
