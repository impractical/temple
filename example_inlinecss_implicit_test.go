package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type InlineCSSImplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type InlineCSSImplicitDependencyHomePage struct {
	Layout    InlineCSSImplicitDependencyLayout
	User      string
	FontColor string
}

func (InlineCSSImplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h InlineCSSImplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (InlineCSSImplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h InlineCSSImplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (InlineCSSImplicitDependencyHomePage) EmbedCSS(_ context.Context) []temple.CSSInline {
	return []temple.CSSInline{
		// because they're all declared in a single slice like this, an
		// implicit constraint will be created that a must be loaded
		// before b, and b must be loaded before c.
		{TemplatePath: "a.css.tmpl"},
		{TemplatePath: "b.css.tmpl"},
		{TemplatePath: "c.css.tmpl"},
	}
}

type InlineCSSImplicitDependencyLayout struct {
}

func (b InlineCSSImplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (InlineCSSImplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_inlineCSSWithImplicitDependency() {
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
		"a.css.tmpl": "body { font: {{ .Page.FontColor }}; }",
		"b.css.tmpl": "html { margin: 0; }",
		"c.css.tmpl": "html { padding: 0; }",
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := InlineCSSImplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := InlineCSSImplicitDependencyHomePage{
		Layout:    InlineCSSImplicitDependencyLayout{},
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
	// <style>
	// html { margin: 0; }
	// </style>
	// <style>
	// html { padding: 0; }
	// </style>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.</body>
	// </html>
}
