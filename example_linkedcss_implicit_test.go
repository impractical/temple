package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type LinkedCSSImplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type LinkedCSSImplicitDependencyHomePage struct {
	Layout    LinkedCSSImplicitDependencyLayout
	User      string
	FontColor string
}

func (LinkedCSSImplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h LinkedCSSImplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (LinkedCSSImplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h LinkedCSSImplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (LinkedCSSImplicitDependencyHomePage) LinkCSS(_ context.Context) []temple.CSSLink {
	return []temple.CSSLink{
		// because they're all declared in a single slice like this, an
		// implicit constraint will be created that a must be loaded
		// before b, and b must be loaded before c.
		{Href: "https://example.com/a.css"},
		{Href: "https://example.com/b.css"},
		{Href: "https://example.com/c.css"},
	}
}

type LinkedCSSImplicitDependencyLayout struct {
}

func (b LinkedCSSImplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (LinkedCSSImplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_linkedCSSWithImplicitDependency() {
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

	site := LinkedCSSImplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := LinkedCSSImplicitDependencyHomePage{
		Layout:    LinkedCSSImplicitDependencyLayout{},
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
