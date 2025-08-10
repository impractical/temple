package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type InlineJSSite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type InlineJSHomePage struct {
	Layout InlineJSLayout
	User   string
}

func (InlineJSHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h InlineJSHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (InlineJSHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h InlineJSHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

func (InlineJSHomePage) EmbedJS(_ context.Context) []temple.JSInline {
	return []temple.JSInline{
		{TemplatePath: "home.js.tmpl"},
		{TemplatePath: "footer.js.tmpl", PlaceInFooter: true},
	}
}

type InlineJSLayout struct {
}

func (b InlineJSLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (InlineJSLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_inlineJS() {
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
		"home.js.tmpl":   `alert("hello, {{ .Page.User }}");`,
		"footer.js.tmpl": `document.write("this was called from the footer");`,
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := InlineJSSite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := InlineJSHomePage{
		Layout: InlineJSLayout{},
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
	// </head>
	// 	<body>
	// 		Hello, Visitor. This is my home page.<script>
	// document.write("this was called from the footer");
	// </script>
	// </body>
	// </html>
}
