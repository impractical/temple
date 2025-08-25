package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type InlineJSExplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type InlineJSExplicitDependencyHomePage struct {
	Layout InlineJSExplicitDependencyLayout
	User   string
}

func (InlineJSExplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h InlineJSExplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (InlineJSExplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h InlineJSExplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (InlineJSExplicitDependencyHomePage) EmbedJS(_ context.Context) []temple.JSInline {
	// we want all our dependencies to render before the dependencies from
	// InlineJSExplicitDependencyLayout.
	//
	// we also want header.c.js.tmpl to render before header.a.js.tmpl and
	// footer.c.js.tmpl before footer.a.js.tmpl.
	beforeGlobalInlines := func(_ context.Context, other temple.JSInline) temple.ResourceRelationship {
		if other.TemplatePath == "global-header.js.tmpl" {
			return temple.ResourceRelationshipBefore
		}
		if other.TemplatePath == "global-footer.js.tmpl" {
			return temple.ResourceRelationshipBefore
		}
		return temple.ResourceRelationshipNeutral
	}
	beforeGlobalLinks := func(_ context.Context, other temple.JSLink) temple.ResourceRelationship {
		if other.Src == "https://example.com/global/a.js" {
			return temple.ResourceRelationshipBefore
		}
		if other.Src == "https://example.com/global/b.js" {
			return temple.ResourceRelationshipBefore
		}
		return temple.ResourceRelationshipNeutral
	}
	return []temple.JSInline{
		{
			TemplatePath:               "header.a.js.tmpl",
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator:   beforeGlobalLinks,
		},
		{
			TemplatePath:               "footer.a.js.tmpl",
			PlaceInFooter:              true,
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator:   beforeGlobalLinks,
		},
		{
			TemplatePath:               "header.b.js.tmpl",
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator:   beforeGlobalLinks,
		},
		{
			TemplatePath:               "footer.b.js.tmpl",
			PlaceInFooter:              true,
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator:   beforeGlobalLinks,
		},
		{
			TemplatePath: "header.c.js.tmpl",
			JSInlineRelationCalculator: func(ctx context.Context, other temple.JSInline) temple.ResourceRelationship {
				if other.TemplatePath == "header.a.js.tmpl" {
					return temple.ResourceRelationshipBefore
				}
				return beforeGlobalInlines(ctx, other)
			},
			JSLinkRelationCalculator: beforeGlobalLinks,
		},
		{
			TemplatePath:  "footer.c.js.tmpl",
			PlaceInFooter: true,
			JSInlineRelationCalculator: func(ctx context.Context, other temple.JSInline) temple.ResourceRelationship {
				if other.TemplatePath == "footer.a.js.tmpl" {
					return temple.ResourceRelationshipBefore
				}
				return beforeGlobalInlines(ctx, other)
			},
			JSLinkRelationCalculator: beforeGlobalLinks,
		},
	}
}

type InlineJSExplicitDependencyLayout struct {
}

func (b InlineJSExplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (InlineJSExplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func (InlineJSExplicitDependencyLayout) EmbedJS(_ context.Context) []temple.JSInline {
	return []temple.JSInline{
		{TemplatePath: "global-header.js.tmpl"},
		{TemplatePath: "global-footer.js.tmpl", PlaceInFooter: true},
	}
}

func (InlineJSExplicitDependencyLayout) LinkJS(_ context.Context) []temple.JSLink {
	return []temple.JSLink{
		{Src: "https://example.com/global/a.js"},
		{Src: "https://example.com/global/b.js"},
	}
}

func ExampleRender_inlineJSWithExplicitDependency() {
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
		"header.a.js.tmpl":      `alert("hello, {{ .Page.User }}");`,
		"footer.a.js.tmpl":      `document.write("this was called from the footer");`,
		"header.b.js.tmpl":      `console.log("header.b.js loaded");`,
		"footer.b.js.tmpl":      `document.write("footer.b.js loaded");`,
		"header.c.js.tmpl":      `console.log("header.c.js loaded");`,
		"footer.c.js.tmpl":      `document.write("footer.c.js loaded");`,
		"global-header.js.tmpl": `console.log("global header loaded");`,
		"global-footer.js.tmpl": `console.log("global footer loaded");`,
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := InlineJSExplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := InlineJSExplicitDependencyHomePage{
		Layout: InlineJSExplicitDependencyLayout{},
		User:   "Visitor",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><script>
	// console.log("header.b.js loaded");
	// </script>
	// <script>
	// console.log("header.c.js loaded");
	// </script>
	// <script>
	// alert("hello, Visitor");
	// </script>
	// <script src="https://example.com/global/a.js"></script>
	// <script src="https://example.com/global/b.js"></script>
	// <script>
	// console.log("global header loaded");
	// </script>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.<script>
	// document.write("footer.b.js loaded");
	// </script>
	// <script>
	// document.write("footer.c.js loaded");
	// </script>
	// <script>
	// document.write("this was called from the footer");
	// </script>
	// <script>
	// console.log("global footer loaded");
	// </script>
	// </body>
	// </html>
}
