package temple_test

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"impractical.co/temple"
)

type CachedSiteLayout struct{}

func (CachedSiteLayout) Templates(_ context.Context) []string {
	return []string{"base.tmpl"}
}

type CachedSiteFoo struct{}

func (CachedSiteFoo) Templates(_ context.Context) []string {
	return []string{"base.tmpl", "foo.tmpl"}
}

func (CachedSiteFoo) Key(_ context.Context) string {
	return "foo.tmpl"
}

func (CachedSiteFoo) ExecutedTemplate(_ context.Context) string {
	return "base.tmpl"
}

type CachedSiteBar struct {
	IncludeBaz bool
}

func (bar CachedSiteBar) Templates(_ context.Context) []string {
	templates := []string{"base.tmpl", "bar.tmpl"}
	if bar.IncludeBaz {
		templates = append(templates, "baz.tmpl")
	}
	return templates
}

func (CachedSiteBar) Key(_ context.Context) string {
	return "bar.tmpl"
}

func (CachedSiteBar) ExecutedTemplate(_ context.Context) string {
	return "base.tmpl"
}

func TestCachedSite(t *testing.T) {
	t.Parallel()

	ctx := temple.LoggingContext(context.Background(), slog.Default())
	templateFS := fstest.MapFS(map[string]*fstest.MapFile{
		"foo.tmpl": {
			Data:    []byte(`{{ define "template_name" }}foo.tmpl{{ end }}`),
			Mode:    0777,
			ModTime: time.Now(),
		},
		"bar.tmpl": {
			Data:    []byte(`{{ define "template_name" }}bar.tmpl{{ if .Page.IncludeBaz }} {{ block "variable_include" . }}{{ end }}{{ end }}{{ end }}`),
			Mode:    0777,
			ModTime: time.Now(),
		},
		"baz.tmpl": {
			Data:    []byte(`{{ define "variable_include" }}included baz.tmpl{{ end }}`),
			Mode:    0777,
			ModTime: time.Now(),
		},
		"base.tmpl": {
			Data:    []byte(`{{ block "template_name" . }}base.tmpl{{ end }}`),
			Mode:    0777,
			ModTime: time.Now(),
		},
	})
	site := temple.NewCachedSite(templateFS)
	renderChangeAndRerender(t, ctx, templateFS, CachedSiteFoo{}, site, "foo.tmpl", "foo.tmpl")
	renderChangeAndRerender(t, ctx, templateFS, CachedSiteBar{}, site, "bar.tmpl", "bar.tmpl")
	renderChangeAndRerender(t, ctx, templateFS, CachedSiteBar{IncludeBaz: true}, site, "bar.tmpl", "bar.tmpl included baz.tmpl")
}

func renderChangeAndRerender(t *testing.T, ctx context.Context, fs fstest.MapFS, page temple.Page, site temple.Site, file, expected string) {
	var out bytes.Buffer
	temple.Render(ctx, &out, site, page)
	if output := out.String(); output != expected {
		t.Errorf("Expected to get %q, got %q", expected, output)
	}
	out.Reset()
	oldData := slices.Clone(fs[file].Data)
	fs[file].Data = []byte(strings.ReplaceAll(string(fs[file].Data), expected, "changed-"+expected))
	temple.Render(ctx, &out, site, page)
	if output := out.String(); output != expected {
		t.Errorf("Expected to get %q after modifying underlying data, got %q", expected, output)
	}
	fs[file].Data = oldData
}
