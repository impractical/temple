package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type LinkedJSDisableImplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type LinkedJSDisableImplicitDependencyHomePage struct {
	Layout LinkedJSDisableImplicitDependencyLayout
	User   string
}

func (LinkedJSDisableImplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h LinkedJSDisableImplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (LinkedJSDisableImplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h LinkedJSDisableImplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (LinkedJSDisableImplicitDependencyHomePage) LinkJS(_ context.Context) []temple.JSLink {
	return []temple.JSLink{
		// because these are all declared in a single component,
		// they'll have an implicit ordering when rendered. JavaScript
		// resources get divided into two groups; those with
		// PlaceInFooter true and those with PlaceInFooter false, then
		// ordered by their constraints. So header.b.js will
		// always be rendered before header.a.js, which will
		// always be rendered before header.c.js. Likewise,
		// footer.b.js will always be rendered before
		// footer.a.js, which will always be rendered before
		// footer.c.js.
		//
		// However, because DisableImplicitOrdering is set on
		// header.b.js and footer.b.js, header.a.js will have no
		// dependency on header.b.js, and footer.a.js will have no
		// dependency on footer.b.js, and the normal lexicographical
		// ordering will decide to put header.a.js and footer.a.js
		// first.
		{Src: "https://example.com/header.b.js", DisableImplicitOrdering: true},
		{Src: "https://example.com/footer.b.js", PlaceInFooter: true, DisableImplicitOrdering: true},
		{Src: "https://example.com/header.a.js"},
		{Src: "https://example.com/footer.a.js", PlaceInFooter: true},
		{Src: "https://example.com/header.c.js"},
		{Src: "https://example.com/footer.c.js", PlaceInFooter: true},
	}
}

type LinkedJSDisableImplicitDependencyLayout struct {
}

func (b LinkedJSDisableImplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (LinkedJSDisableImplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_linkedJSWithDisableImplicitDependency() {
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

	site := LinkedJSDisableImplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := LinkedJSDisableImplicitDependencyHomePage{
		Layout: LinkedJSDisableImplicitDependencyLayout{},
		User:   "Visitor",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><script src="https://example.com/header.a.js"></script>
	// <script src="https://example.com/header.b.js"></script>
	// <script src="https://example.com/header.c.js"></script>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.<script src="https://example.com/footer.a.js"></script>
	// <script src="https://example.com/footer.b.js"></script>
	// <script src="https://example.com/footer.c.js"></script>
	// </body>
	// </html>
}
