package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type InlineCSSDisableImplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type InlineCSSDisableImplicitDependencyHomePage struct {
	Layout    InlineCSSDisableImplicitDependencyLayout
	User      string
	FontColor string
}

func (InlineCSSDisableImplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h InlineCSSDisableImplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (InlineCSSDisableImplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h InlineCSSDisableImplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (InlineCSSDisableImplicitDependencyHomePage) EmbedCSS(_ context.Context) []temple.CSSInline {
	return []temple.CSSInline{
		// because they're all declared in a single slice like this, an
		// implicit constraint will be created that b must be loaded
		// before a, and a must be loaded before c.
		//
		// However, because DisableImplicitOrdering is set on b, a will
		// have no dependency on b, and the normal lexicographical
		// ordering will decide to put a first.
		{TemplatePath: "b.css.tmpl", DisableImplicitOrdering: true},
		{TemplatePath: "a.css.tmpl"},
		{TemplatePath: "c.css.tmpl"},
	}
}

type InlineCSSDisableImplicitDependencyLayout struct {
}

func (b InlineCSSDisableImplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (InlineCSSDisableImplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_inlineCSSWithDisableImplicitDependency() {
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

	site := InlineCSSDisableImplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := InlineCSSDisableImplicitDependencyHomePage{
		Layout:    InlineCSSDisableImplicitDependencyLayout{},
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
