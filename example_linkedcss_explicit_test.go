package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type LinkedCSSExplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type LinkedCSSExplicitDependencyHomePage struct {
	Layout LinkedCSSExplicitDependencyLayout
	User   string
}

func (LinkedCSSExplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h LinkedCSSExplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (LinkedCSSExplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h LinkedCSSExplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (LinkedCSSExplicitDependencyHomePage) LinkCSS(_ context.Context) []temple.CSSLink {
	// we want all our dependencies to render before the
	// dependencies from LinkedCSSExplicitDependencyLayout.
	//
	// we also want https://example.com/c.css to render before
	// https://example.com/a.css.
	beforeGlobalInlines := func(_ context.Context, other temple.CSSInline) temple.ResourceRelationship {
		if other.TemplatePath == "global.css.tmpl" {
			return temple.ResourceRelationshipBefore
		}
		return temple.ResourceRelationshipNeutral
	}
	beforeGlobalLinks := func(_ context.Context, other temple.CSSLink) temple.ResourceRelationship {
		if other.Href == "https://example.com/global/a.css" {
			return temple.ResourceRelationshipBefore
		}
		return temple.ResourceRelationshipNeutral
	}
	return []temple.CSSLink{
		{Href: "https://example.com/a.css", CSSInlineRelationCalculator: beforeGlobalInlines, CSSLinkRelationCalculator: beforeGlobalLinks},
		{Href: "https://example.com/b.css", CSSInlineRelationCalculator: beforeGlobalInlines, CSSLinkRelationCalculator: beforeGlobalLinks},
		{Href: "https://example.com/c.css", CSSInlineRelationCalculator: beforeGlobalInlines, CSSLinkRelationCalculator: func(ctx context.Context, other temple.CSSLink) temple.ResourceRelationship {
			if other.Href == "https://example.com/a.css" {
				return temple.ResourceRelationshipBefore
			}
			return beforeGlobalLinks(ctx, other)
		}},
	}
}

type LinkedCSSExplicitDependencyLayout struct {
}

func (b LinkedCSSExplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (LinkedCSSExplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func (LinkedCSSExplicitDependencyLayout) EmbedCSS(_ context.Context) []temple.CSSInline {
	return []temple.CSSInline{
		{TemplatePath: "global.css.tmpl"},
	}
}

func (LinkedCSSExplicitDependencyLayout) LinkCSS(_ context.Context) []temple.CSSLink {
	return []temple.CSSLink{
		{Href: "https://example.com/global/a.css"},
	}
}

func ExampleRender_linkedCSSWithExplicitDependency() {
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
		"global.css.tmpl": "body { margin: 0; }",
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := LinkedCSSExplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := LinkedCSSExplicitDependencyHomePage{
		Layout: LinkedCSSExplicitDependencyLayout{},
		User:   "Visitor",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><link href="https://example.com/b.css">
	// <link href="https://example.com/c.css">
	// <link href="https://example.com/a.css">
	// <link href="https://example.com/global/a.css">
	// <style>
	// body { margin: 0; }
	// </style>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.</body>
	// </html>
}
