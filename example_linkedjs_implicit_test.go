package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type LinkedJSImplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type LinkedJSImplicitDependencyHomePage struct {
	Layout LinkedJSImplicitDependencyLayout
	User   string
}

func (LinkedJSImplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h LinkedJSImplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (LinkedJSImplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h LinkedJSImplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (LinkedJSImplicitDependencyHomePage) LinkJS(_ context.Context) []temple.JSLink {
	return []temple.JSLink{
		// because these are all declared in a single component,
		// they'll have an implicit ordering when rendered. JavaScript
		// resources get divided into two groups; those with
		// PlaceInFooter true and those with PlaceInFooter false, then
		// ordered by their constraints. So header.a.js will
		// always be rendered before header.b.js, which will
		// always be rendered before header.c.js. Likewise,
		// footer.a.js will always be rendered before
		// footer.b.js, which will always be rendered before
		// footer.c.js.
		{Src: "https://example.com/header.a.js"},
		{Src: "https://example.com/footer.a.js", PlaceInFooter: true},
		{Src: "https://example.com/header.b.js"},
		{Src: "https://example.com/footer.b.js", PlaceInFooter: true},
		{Src: "https://example.com/header.c.js"},
		{Src: "https://example.com/footer.c.js", PlaceInFooter: true},
	}
}

type LinkedJSImplicitDependencyLayout struct {
}

func (b LinkedJSImplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (LinkedJSImplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_linkedJSWithImplicitDependency() {
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

	site := LinkedJSImplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := LinkedJSImplicitDependencyHomePage{
		Layout: LinkedJSImplicitDependencyLayout{},
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
