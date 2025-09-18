package temple_test

import (
	"context"
	"html/template"
	"log/slog"
	"os"
	"strings"

	"impractical.co/temple"
)

type FuncMapSite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

func (site FuncMapSite) FuncMap(_ context.Context) template.FuncMap {
	// these functions will be available to all components and pages
	return template.FuncMap{
		"applesAndOranges": func(in []string) []string {
			for pos, fruit := range in {
				if strings.ToLower(fruit) == "apples" {
					in[pos] = "oranges"
				}
			}
			return in
		},
	}
}

type FuncMapHomePage struct {
	Name   string
	Fruits []string
	Layout FuncMapLayout
}

func (FuncMapHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (page FuncMapHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		page.Layout,
	}
}

func (FuncMapHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (page FuncMapHomePage) ExecutedTemplate(_ context.Context) string {
	return page.Layout.BaseTemplate()
}

func (FuncMapHomePage) FuncMap(_ context.Context) template.FuncMap {
	return template.FuncMap{
		// this function is only available to the homepage
		"humanize": func(input []string) string {
			if len(input) < 1 {
				return ""
			}
			if len(input) < 2 {
				return strings.Join(input, " and ")
			}
			input[len(input)-1] = "and " + input[len(input)-1]
			return strings.Join(input, ", ")
		},
	}
}

type FuncMapLayout struct {
}

func (layout FuncMapLayout) Templates(_ context.Context) []string {
	return []string{layout.BaseTemplate()}
}

func (FuncMapLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func ExampleRender_funcMaps() {
	// normally you'd use something like embed.FS or os.DirFS for this
	// for example purposes, we're just hardcoding values
	var templates = staticFS{
		"home.html.tmpl": `{{ define "body" }}Hello, {{ .Page.Name }}. This is my home page. I like {{ humanize (applesAndOranges .Page.Fruits) }}.{{ end }}`,
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

	site := FuncMapSite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := FuncMapHomePage{
		Name:   "Visitor",
		Fruits: []string{"apples", "bananas", "oranges"},
		Layout: FuncMapLayout{},
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title></head>
	// 	<body>
	// 		Hello, Visitor. This is my home page. I like oranges, bananas, and oranges.</body>
	// </html>
}
