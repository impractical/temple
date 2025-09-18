package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type InlineCSSExplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type InlineCSSExplicitDependencyHomePage struct {
	Layout    InlineCSSExplicitDependencyLayout
	User      string
	FontColor string
}

func (InlineCSSExplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h InlineCSSExplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (InlineCSSExplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h InlineCSSExplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (InlineCSSExplicitDependencyHomePage) EmbedCSS(_ context.Context) []temple.CSSInline {
	// we want all our dependencies to render before the
	// dependencies from InlineCSSExplicitDependencyLayout.
	//
	// we also want c.css.tmpl to render before a.css.tmpl.
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
	return []temple.CSSInline{
		{TemplatePath: "a.css.tmpl", CSSInlineRelationCalculator: beforeGlobalInlines, CSSLinkRelationCalculator: beforeGlobalLinks},
		{TemplatePath: "b.css.tmpl", CSSInlineRelationCalculator: beforeGlobalInlines, CSSLinkRelationCalculator: beforeGlobalLinks},
		{TemplatePath: "c.css.tmpl", CSSLinkRelationCalculator: beforeGlobalLinks, CSSInlineRelationCalculator: func(ctx context.Context, other temple.CSSInline) temple.ResourceRelationship {
			if other.TemplatePath == "a.css.tmpl" {
				return temple.ResourceRelationshipBefore
			}
			return beforeGlobalInlines(ctx, other)
		}},
	}
}

type InlineCSSExplicitDependencyLayout struct {
}

func (b InlineCSSExplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (InlineCSSExplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func (InlineCSSExplicitDependencyLayout) EmbedCSS(_ context.Context) []temple.CSSInline {
	return []temple.CSSInline{
		{TemplatePath: "global.css.tmpl"},
	}
}

func (InlineCSSExplicitDependencyLayout) LinkCSS(_ context.Context) []temple.CSSLink {
	return []temple.CSSLink{
		{Href: "https://example.com/global/a.css"},
	}
}

func ExampleRender_inlineCSSWithExplicitDependency() {
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
		"a.css.tmpl":      "body { font: {{ .Page.FontColor }}; }",
		"b.css.tmpl":      "html { margin: 0; }",
		"c.css.tmpl":      "html { padding: 0; }",
		"global.css.tmpl": "body { margin: 0; }",
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := InlineCSSExplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := InlineCSSExplicitDependencyHomePage{
		Layout:    InlineCSSExplicitDependencyLayout{},
		User:      "Visitor",
		FontColor: "red",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><style>
	// html { margin: 0; }
	// </style>
	// <style>
	// html { padding: 0; }
	// </style>
	// <style>
	// body { font: red; }
	// </style>
	// <link href="https://example.com/global/a.css">
	// <style>
	// body { margin: 0; }
	// </style>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.</body>
	// </html>
}
