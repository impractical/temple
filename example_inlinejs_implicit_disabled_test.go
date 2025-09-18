package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type InlineJSDisableImplicitDependencySite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type InlineJSDisableImplicitDependencyHomePage struct {
	Layout InlineJSDisableImplicitDependencyLayout
	User   string
}

func (InlineJSDisableImplicitDependencyHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h InlineJSDisableImplicitDependencyHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (InlineJSDisableImplicitDependencyHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h InlineJSDisableImplicitDependencyHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (InlineJSDisableImplicitDependencyHomePage) EmbedJS(_ context.Context) []temple.JSInline {
	return []temple.JSInline{
		// because these are all declared in a single component,
		// they'll have an implicit ordering when rendered. JavaScript
		// resources get divided into two groups; those with
		// PlaceInFooter true and those with PlaceInFooter false, then
		// ordered by their constraints. So header.b.js.tmpl will
		// always be rendered before header.a.js.tmpl, which will
		// always be rendered before header.c.js.tmpl. Likewise,
		// footer.b.js.tmpl will always be rendered before
		// footer.a.js.tmpl, which will always be rendered before
		// footer.c.js.tmpl.
		//
		// However, because DisableImplicitOrdering is set on
		// header.b.js.tmpl and footer.b.js.tmpl, header.a.js.tmpl will
		// have no dependency on header.b.js.tmpl, and footer.a.js.tmpl
		// will have no dependency on footer.b.js.tmpl, and the normal
		// lexicographical ordering will decide to put header.a.js.tmpl
		// and footer.a.js.tmpl first.
		{TemplatePath: "header.b.js.tmpl", DisableImplicitOrdering: true},
		{TemplatePath: "footer.b.js.tmpl", PlaceInFooter: true, DisableImplicitOrdering: true},
		{TemplatePath: "header.a.js.tmpl"},
		{TemplatePath: "footer.a.js.tmpl", PlaceInFooter: true},
		{TemplatePath: "header.c.js.tmpl"},
		{TemplatePath: "footer.c.js.tmpl", PlaceInFooter: true},
	}
}

type InlineJSDisableImplicitDependencyLayout struct {
}

func (b InlineJSDisableImplicitDependencyLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (InlineJSDisableImplicitDependencyLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_inlineJSWithDisableImplicitDependency() {
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
		"header.a.js.tmpl": `alert("hello, {{ .Page.User }}");`,
		"footer.a.js.tmpl": `document.write("this was called from the footer");`,
		"header.b.js.tmpl": `console.log("header.b.js loaded");`,
		"footer.b.js.tmpl": `document.write("footer.b.js loaded");`,
		"header.c.js.tmpl": `console.log("header.c.js loaded");`,
		"footer.c.js.tmpl": `document.write("footer.c.js loaded");`,
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := InlineJSDisableImplicitDependencySite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := InlineJSDisableImplicitDependencyHomePage{
		Layout: InlineJSDisableImplicitDependencyLayout{},
		User:   "Visitor",
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title><script>
	// alert("hello, Visitor");
	// </script>
	// <script>
	// console.log("header.b.js loaded");
	// </script>
	// <script>
	// console.log("header.c.js loaded");
	// </script>
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.<script>
	// document.write("this was called from the footer");
	// </script>
	// <script>
	// document.write("footer.b.js loaded");
	// </script>
	// <script>
	// document.write("footer.c.js loaded");
	// </script>
	// </body>
	// </html>
}
