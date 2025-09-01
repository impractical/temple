package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type LinkedJSExplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type LinkedJSExplicitDependencyHomePage struct {
	Layout LinkedJSExplicitDependencyLayout
	User   string
}

func (LinkedJSExplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h LinkedJSExplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (LinkedJSExplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h LinkedJSExplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (LinkedJSExplicitDependencyHomePage) LinkJS(_ context.Context) []temple.JSLink {
	// we want all our dependencies to render before the dependencies from
	// LinkedJSExplicitDependencyLayout.
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
	return []temple.JSLink{
		{
			Src:                        "https://example.com/header.a.js",
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator:   beforeGlobalLinks,
		},
		{
			Src:                        "https://example.com/footer.a.js",
			PlaceInFooter:              true,
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator:   beforeGlobalLinks,
		},
		{
			Src:                        "https://example.com/header.b.js",
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator:   beforeGlobalLinks,
		},
		{
			Src:                        "https://example.com/footer.b.js",
			PlaceInFooter:              true,
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator:   beforeGlobalLinks,
		},
		{
			Src:                        "https://example.com/header.c.js",
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator: func(ctx context.Context, other temple.JSLink) temple.ResourceRelationship {
				if other.Src == "https://example.com/header.a.js" {
					return temple.ResourceRelationshipBefore
				}
				return beforeGlobalLinks(ctx, other)
			},
		},
		{
			Src:                        "https://example.com/footer.c.js",
			PlaceInFooter:              true,
			JSInlineRelationCalculator: beforeGlobalInlines,
			JSLinkRelationCalculator: func(ctx context.Context, other temple.JSLink) temple.ResourceRelationship {
				if other.Src == "https://example.com/footer.a.js" {
					return temple.ResourceRelationshipBefore
				}
				return beforeGlobalLinks(ctx, other)
			},
		},
	}
}

type LinkedJSExplicitDependencyLayout struct {
}

func (b LinkedJSExplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (LinkedJSExplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func (LinkedJSExplicitDependencyLayout) EmbedJS(_ context.Context) []temple.JSInline {
	return []temple.JSInline{
		{TemplatePath: "global-header.js.tmpl"},
		{TemplatePath: "global-footer.js.tmpl", PlaceInFooter: true},
	}
}

func (LinkedJSExplicitDependencyLayout) LinkJS(_ context.Context) []temple.JSLink {
	return []temple.JSLink{
		{Src: "https://example.com/global/a.js"},
		{Src: "https://example.com/global/b.js"},
	}
}

func ExampleRender_linkedJSWithExplicitDependency() {
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
		"global-header.js.tmpl": `console.log("global header loaded");`,
		"global-footer.js.tmpl": `console.log("global footer loaded");`,
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := LinkedJSExplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := LinkedJSExplicitDependencyHomePage{
		Layout: LinkedJSExplicitDependencyLayout{},
		User:   "Visitor",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><script src="https://example.com/header.b.js"></script>
	// <script src="https://example.com/header.c.js"></script>
	// <script src="https://example.com/header.a.js"></script>
	// <script src="https://example.com/global/a.js"></script>
	// <script src="https://example.com/global/b.js"></script>
	// <script>
	// console.log("global header loaded");
	// </script>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.<script src="https://example.com/footer.b.js"></script>
	// <script src="https://example.com/footer.c.js"></script>
	// <script src="https://example.com/footer.a.js"></script>
	// <script>
	// console.log("global footer loaded");
	// </script>
	// </body>
	// </html>
}
