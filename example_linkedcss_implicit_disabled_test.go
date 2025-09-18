package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type LinkedCSSDisableImplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type LinkedCSSDisableImplicitDependencyHomePage struct {
	Layout    LinkedCSSDisableImplicitDependencyLayout
	User      string
	FontColor string
}

func (LinkedCSSDisableImplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h LinkedCSSDisableImplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (LinkedCSSDisableImplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h LinkedCSSDisableImplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (LinkedCSSDisableImplicitDependencyHomePage) LinkCSS(_ context.Context) []temple.CSSLink {
	return []temple.CSSLink{
		// because they're all declared in a single slice like this, an
		// implicit constraint will be created that b must be loaded
		// before a, and a must be loaded before c.
		//
		// However, because DisableImplicitOrdering is set on b, a will
		// have no dependency on b, and the normal lexicographical
		// ordering will decide to put a first.
		{Href: "https://example.com/b.css", DisableImplicitOrdering: true},
		{Href: "https://example.com/a.css"},
		{Href: "https://example.com/c.css"},
	}
}

type LinkedCSSDisableImplicitDependencyLayout struct {
}

func (b LinkedCSSDisableImplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (LinkedCSSDisableImplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_linkedCSSWithDisableImplicitDependency() {
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

	site := LinkedCSSDisableImplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := LinkedCSSDisableImplicitDependencyHomePage{
		Layout:    LinkedCSSDisableImplicitDependencyLayout{},
		User:      "Visitor",
		FontColor: "red",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><link href="https://example.com/a.css">
	// <link href="https://example.com/b.css">
	// <link href="https://example.com/c.css">
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.</body>
	// </html>
}
