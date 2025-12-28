package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type DuplicateResourcesSite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type DuplicateResourcesHomePage struct {
	Name   string
	Layout DuplicateResourcesLayout
}

func (DuplicateResourcesHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (page DuplicateResourcesHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		page.Layout,
	}
}

func (DuplicateResourcesHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (page DuplicateResourcesHomePage) ExecutedTemplate(_ context.Context) string {
	return page.Layout.BaseTemplate()
}

func (DuplicateResourcesHomePage) LinkCSS(_ context.Context) []temple.CSSLink {
	return []temple.CSSLink{
		{Href: "https://example.com/b.css", Type: "text/css", Rel: "stylesheet"},
		{Href: "https://example.com/c.css", Type: "text/css", Rel: "stylesheet"},
	}
}

func (DuplicateResourcesHomePage) EmbedCSS(_ context.Context) []temple.CSSInline {
	return []temple.CSSInline{
		{TemplatePath: "b.inline.css"},
		{TemplatePath: "c.inline.css"},
	}
}

func (DuplicateResourcesHomePage) LinkJS(_ context.Context) []temple.JSLink {
	return []temple.JSLink{
		{Src: "https://example.com/b.js", Type: "text/javascript"},
		{Src: "https://example.com/c.js", Type: "text/javascript"},
	}
}

func (DuplicateResourcesHomePage) EmbedJS(_ context.Context) []temple.JSInline {
	return []temple.JSInline{
		{TemplatePath: "b.inline.js"},
		{TemplatePath: "c.inline.js"},
	}
}

type DuplicateResourcesLayout struct {
}

func (layout DuplicateResourcesLayout) Templates(_ context.Context) []string {
	return []string{layout.BaseTemplate()}
}

func (DuplicateResourcesLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func (DuplicateResourcesLayout) LinkCSS(_ context.Context) []temple.CSSLink {
	return []temple.CSSLink{
		{Href: "https://example.com/a.css", Type: "text/css", Rel: "stylesheet"},
		{Href: "https://example.com/b.css", Type: "text/css", Rel: "stylesheet"},
	}
}

func (DuplicateResourcesLayout) EmbedCSS(_ context.Context) []temple.CSSInline {
	return []temple.CSSInline{
		{TemplatePath: "a.inline.css"},
		{TemplatePath: "b.inline.css"},
	}
}

func (DuplicateResourcesLayout) LinkJS(_ context.Context) []temple.JSLink {
	return []temple.JSLink{
		{Src: "https://example.com/a.js", Type: "text/javascript"},
		{Src: "https://example.com/b.js", Type: "text/javascript"},
	}
}

func (DuplicateResourcesLayout) EmbedJS(_ context.Context) []temple.JSInline {
	return []temple.JSInline{
		{TemplatePath: "a.inline.js"},
		{TemplatePath: "b.inline.js"},
	}
}

func ExampleRender_duplicateResources() {
	// normally you'd use something like embed.FS or os.DirFS for this
	// for example purposes, we're just hardcoding values
	var templates = staticFS{
		"home.html.tmpl": `{{ define "body" }}Hello, {{ .Page.Name }}. This is my home page.{{ end }}`,
		"a.inline.css":   `body { color: red; }`,
		"b.inline.css":   `html { background-color: black; }`,
		"c.inline.css":   `a { color: yellow; }`,
		"a.inline.js":    `alert('a!');`,
		"b.inline.js":    `alert('b!');`,
		"c.inline.js":    `alert('c!');`,
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

	site := DuplicateResourcesSite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := DuplicateResourcesHomePage{
		Name:   "Visitor",
		Layout: DuplicateResourcesLayout{},
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><link href="https://example.com/a.css" rel="stylesheet" type="text/css">
	// <link href="https://example.com/b.css" rel="stylesheet" type="text/css">
	// <link href="https://example.com/c.css" rel="stylesheet" type="text/css">
	// <style>
	// body { color: red; }
	// </style>
	// <style>
	// html { background-color: black; }
	// </style>
	// <style>
	// a { color: yellow; }
	// </style>
	// <script type="text/javascript" src="https://example.com/a.js"></script>
	// <script type="text/javascript" src="https://example.com/b.js"></script>
	// <script type="text/javascript" src="https://example.com/c.js"></script>
	// <script>
	// alert('a!');
	// </script>
	// <script>
	// alert('b!');
	// </script>
	// <script>
	// alert('c!');
	// </script>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.</body>
	// </html>
}
