package temple_test

import (
	"context"
	"log/slog"
	"os"

	"impractical.co/temple"
)

type DuplicateComponentsSite struct {
	// anonymously embedding a *CachedSite makes MySite a Site implementation
	*temple.CachedSite

	// a configurable title for our site
	Title string
}

type DuplicateComponentsHomePage struct {
	Name   string
	Layout DuplicateComponentsLayout
	Avatar DuplicateComponentsAvatar
}

func (DuplicateComponentsHomePage) Templates(_ context.Context) []string {
	return []string{"home.html.tmpl"}
}

func (page DuplicateComponentsHomePage) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		page.Layout,
		page.Avatar,
	}
}

func (DuplicateComponentsHomePage) Key(_ context.Context) string {
	return "home.html.tmpl"
}

func (page DuplicateComponentsHomePage) ExecutedTemplate(_ context.Context) string {
	return page.Layout.BaseTemplate()
}

type DuplicateComponentsLayout struct {
	Avatar DuplicateComponentsAvatar
}

func (layout DuplicateComponentsLayout) Templates(_ context.Context) []string {
	return []string{layout.BaseTemplate()}
}

func (DuplicateComponentsLayout) BaseTemplate() string {
	return "base.html.tmpl"
}

func (layout DuplicateComponentsLayout) UseComponents(_ context.Context) []temple.Component {
	return []temple.Component{
		layout.Avatar,
	}
}

type DuplicateComponentsAvatar struct {
	URL string
}

func (DuplicateComponentsAvatar) Templates(_ context.Context) []string {
	return []string{"avatar.html.tmpl"}
}

func (DuplicateComponentsAvatar) Key(_ context.Context) string {
	return "avatar.html.tmpl"
}

func ExampleRender_duplicateComponents() {
	// normally you'd use something like embed.FS or os.DirFS for this
	// for example purposes, we're just hardcoding values
	var templates = staticFS{
		"home.html.tmpl":   `{{ define "body" }}Hello, {{ .Page.Name }}. This is my home page. Here is your avatar: {{ block "avatar" .Page.Avatar }}{{ end }}{{ end }}`,
		"avatar.html.tmpl": `{{ define "avatar" }}<img src="{{ .URL }}">{{ end }}`,
		"base.html.tmpl": `
<!doctype html>
<html lang="en">
	<head>
		<title>{{ .Site.Title }}</title>
		{{- .CSS -}}
		{{- .HeaderJS -}}
	</head>
	<body>
		<nav>
			<h1>{{ .Site.Title }}</h1>
			<span class="align-right">{{ block "avatar" .Page.Layout.Avatar }}{{ end }}</span>
		</nav>
		{{ block "body" . }}{{ end }}
		{{- .FooterJS -}}
	</body>
</html>`,
	}

	// usually the context comes from the request, but here we're building it from scratch and adding a logger
	ctx := temple.LoggingContext(context.Background(), slog.Default())

	site := DuplicateComponentsSite{
		CachedSite: temple.NewCachedSite(templates),
		Title:      "My Example Site",
	}
	page := DuplicateComponentsHomePage{
		Name: "Visitor",
		Layout: DuplicateComponentsLayout{
			Avatar: DuplicateComponentsAvatar{
				URL: "https://www.example.com/me.jpg",
			},
		},
		Avatar: DuplicateComponentsAvatar{
			URL: "https://www.example.com/me.jpg",
		},
	}
	temple.Render(ctx, os.Stdout, site, page)

	//Output:
	// <!doctype html>
	// <html lang="en">
	// 	<head>
	// 		<title>My Example Site</title></head>
	// 	<body>
	// 		<nav>
	// 			<h1>My Example Site</h1>
	// 			<span class="align-right"><img src="https://www.example.com/me.jpg"></span>
	// 		</nav>
	// 		Hello, Visitor. This is my home page. Here is your avatar: <img src="https://www.example.com/me.jpg"></body>
	// </html>
}
