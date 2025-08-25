package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type ErrorSite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

func (ErrorSite) ServerErrorPage(_ context.Context) temple.Page {
	return ErrorPage{}
}

type ErrorHomePage struct {
	Name   string
	Layout ErrorLayout
}

func (ErrorHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (h ErrorHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		h.Layout,
	}
}

func (ErrorHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (h ErrorHomePage) ExecutedTemplate(_ context.Context) string {
	return h.Layout.BaseTemplate()
}

type ErrorLayout struct {
}

func (b ErrorLayout) Templates(_ context.Context) []string {
	return []string{b.BaseTemplate()}
}

func (ErrorLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

type ErrorPage struct{}

func (ErrorPage) Templates(_ context.Context) []string {
	return []string{"server_error.html.tmpl"}
}

func (ErrorPage) Key(_ context.Context) string {
	return "server_error.html.tmpl"
}

func (ErrorPage) ExecutedTemplate(_ context.Context) string {
	return "server_error.html.tmpl"
}

func ExampleRender_test() {
	// for example purposes, we're just hardcoding values
	var templates = staticFS{
		"home.html.tmpl": `{{ define "body" }}Hello, {{ .Page.Name }}. This is my home page.{{ end }}`,
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
		<!-- purposefully throw an error here -->
		{{ .ImaginaryData }}
	</body>
</html>`,
		"server_error.html.tmpl": `
<!doctype html>
<html lang="en">
	<head>
		<title>Server Error</title>
	</head>
	<body>
		<h1>Server error</h1>
		<p>Something went wrong, sorry about that. We're working on fixing it now.</p>
	</body>
</html>`,
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := ErrorSite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := ErrorHomePage{
		Name:   "Visitor",
		Layout: ErrorLayout{},
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>Server Error</title>
	// 	</head>
	// 	<body>
	// 		<h1>Server error</h1>
	// 		<p>Something went wrong, sorry about that. We're working on fixing it now.</p>
	// 	</body>
	// </html>
}
