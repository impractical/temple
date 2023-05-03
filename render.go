package temple

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"

	"yall.in"
)

var (
	// ErrNoTemplatePath is returned when a template path is needed, but
	// none are supplied.
	ErrNoTemplatePath = errors.New("need at least one template path")

	// ErrTemplatePatternMatchesNoFiles is returned when a template path is
	// a pattern, but that pattern doesn't match any files.
	ErrTemplatePatternMatchesNoFiles = errors.New("pattern matches no files")
)

// Component is an interface for a UI component that can be rendered to HTML.
type Component interface {
	// TemplatePaths returns a list of filepaths to html/template contents
	// that need to be parsed before the component can be rendered.
	Templates(context.Context) []string
}

// ComponentUser is an interface that a Component can optionally implement to
// list the Components that it relies upon. These Components will automatically
// have the appropriate methods called if they implement any of the optional
// interfaces. A Component that uses other Components but doesn't return them
// with this interface is responsible for including the output of any optional
// interfaces that they implement in its own implementation of those interfaces
// and including their Templates output in its own Templates output.
type ComponentUser interface {
	// UseComponents returns the Components that this Component relies on.
	UseComponents(context.Context) []Component
}

// FuncMapExtender is an interface that Components can fulfill to add to the
// map of functions available to them when rendering.
type FuncMapExtender interface {
	// FuncMap returns an html/template.FuncMap containing all the
	// functions that the Component is adding to the FuncMap.
	FuncMap(context.Context) template.FuncMap
}

// Renderable is an interface for a page that can be passed to Render. It
// defines a single logical page of the application, composed of one or more
// Components. It should contain all the information needed to render the
// Components to HTML.
type Renderable interface {
	Component

	// Key is a unique key to use when caching this page so it doesn't need
	// to be re-parsed. A good key is consistent, but unique per Renderable.
	Key(context.Context) string

	// ExecutedTemplate is the template that needs to actually be executed
	// when rendering the page.
	//
	// This is usually not the template for the Component defining the
	// page; it's usually the "base" template that Component defining the
	// page fills blocks in.
	ExecutedTemplate(context.Context) string
}

// RenderData is the data that is passed to a page when rendering it.
type RenderData[SiteType Site, PageType Renderable] struct {
	// Site is an instance of the Site type, containing all the
	// configuration and information about a Site. This can be used to
	// avoid passing global configuration options to every single page.
	Site SiteType

	// Page is the information for a specific page, embedded in that page's
	// Renderable type.
	Page PageType

	// EmbeddedJS is the result of calling EmbedJS on the Renderable, if
	// the Renderable supports the JSEmbedder interface.
	EmbeddedJS string

	// EmbeddedCSS is the result of calling EmbedCSS on the Renderable, if
	// the Renderable supports the CSSEmbedder interface.
	EmbeddedCSS string

	// LinkedJS is the result of calling LinkJS on the Renderable, if the
	// Renderable supports the JSLinker interface.
	LinkedJS []string

	// LinkedCSS is the result of calling LinkCSS on the Renderable, if the
	// Renderable supports the CSSLinker interface.
	LinkedCSS []string
}

// Render renders the passed Renderable to the Writer. If it can't, a server
// error page is written instead. If the Site implements ServerErrorPager, that
// will be rendered; if not, a simple text page indicating a server error will
// be written.
func Render[SiteType Site, PageType Renderable](ctx context.Context, out io.Writer, site SiteType, page PageType) {
	defer func() {
		// if the ResponseWriter can be closed, let's try to close it
		if closer, ok := out.(io.Closer); ok {
			err := closer.Close()
			// if there's an error closing it, logging it's about all we can do
			if err != nil {
				yall.FromContext(ctx).
					WithError(err).
					Error("error closing response writer")
			}
		}
	}()

	// try to render the page
	err := basicRender(ctx, out, site, page)

	// if there's no error, we're done here
	if err == nil {
		return
	}

	// if there is an error, we now need to try and render a server error
	// page

	// but first we're logging whatever went wrong
	yall.FromContext(ctx).
		WithError(err).
		Error("error rendering page")

	// now let's render the server error page
	if pager, ok := Site(site).(ServerErrorPager); ok {
		err = basicRender(ctx, out, site, pager.ServerErrorPage(ctx))
		if err != nil {
			// if we can't do that, everything's doomed, doomed, doomed
			// just log it and we'll move on
			yall.FromContext(ctx).
				WithError(err).
				Error("error rendering server error page")
		}
		return
	}

	// there's no default server error page, write a server error message
	_, err = out.Write([]byte("Server error."))
	if err != nil {
		yall.FromContext(ctx).
			WithError(err).
			Error("error writing server error message")
	}
}

func basicRender[SiteType Site, PageType Renderable](ctx context.Context, output io.Writer, site SiteType, page PageType) error {
	tmpl, err := getTemplate(ctx, site, page)
	if err != nil {
		return err
	}

	data := RenderData[SiteType, PageType]{
		Site:        site,
		Page:        page,
		EmbeddedJS:  getComponentJSEmbeds(ctx, page),
		LinkedJS:    getComponentJSLinks(ctx, page),
		EmbeddedCSS: getComponentCSSEmbeds(ctx, page),
		LinkedCSS:   getComponentCSSLinks(ctx, page),
	}

	executed := page.ExecutedTemplate(ctx)
	err = tmpl.ExecuteTemplate(output, executed, data)
	if err != nil {
		return fmt.Errorf("error executing template %q for %T: %w", executed, page, err)
	}
	return nil
}

func getTemplate(ctx context.Context, site Site, page Renderable) (*template.Template, error) {
	key := page.Key(ctx)
	if cache, ok := site.(TemplateCacher); ok {
		cached := cache.GetCachedTemplate(ctx, key)
		if cached != nil {
			return cached, nil
		}
	}
	tmplPaths := getComponentTemplatePaths(ctx, page)
	if len(tmplPaths) < 1 {
		return nil, fmt.Errorf("error rendering %T: %w", page, ErrNoTemplatePath)
	}
	funcMap := getComponentFuncMap(ctx, site, page)
	parsed, err := parseTemplates(site.TemplateDir(ctx), funcMap, tmplPaths...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates %v for page %T: %w", page, tmplPaths, err)
	}
	if cache, ok := site.(TemplateCacher); ok {
		cache.SetCachedTemplate(ctx, key, parsed)
	}
	return parsed, nil
}

func getRecursiveComponents(ctx context.Context, component Component) []Component {
	results := []Component{component}

	if uses, ok := component.(ComponentUser); ok {
		children := uses.UseComponents(ctx)
		for _, child := range children {
			results = append(results, getRecursiveComponents(ctx, child)...)
		}
	}
	return results
}

func getComponentTemplatePaths(ctx context.Context, component Component) []string {
	var results []string
	seen := map[string]struct{}{}
	components := getRecursiveComponents(ctx, component)
	for _, comp := range components {
		paths := comp.Templates(ctx)
		for _, path := range paths {
			if _, ok := seen[path]; !ok {
				results = append(results, path)
				seen[path] = struct{}{}
			}
		}
	}
	return results
}

func getComponentFuncMap(ctx context.Context, site Site, component Component) template.FuncMap {
	results := template.FuncMap{}
	if fm, ok := site.(FuncMapExtender); ok {
		results = mergeFuncMaps(results, fm.FuncMap(ctx))
	}
	components := getRecursiveComponents(ctx, component)
	for _, comp := range components {
		fm, ok := comp.(FuncMapExtender)
		if !ok {
			continue
		}
		results = mergeFuncMaps(results, fm.FuncMap(ctx))
	}
	return results
}

func parseTemplates(fsys fs.FS, funcs template.FuncMap, patterns ...string) (*template.Template, error) {
	var files []string
	for _, pattern := range patterns {
		list, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("error listing files for %q: %w", pattern, err)
		}
		if len(list) < 1 {
			return nil, fmt.Errorf("error parsing %q: %w", pattern, ErrTemplatePatternMatchesNoFiles)
		}
		files = append(files, list...)
	}
	if len(files) < 1 {
		return nil, ErrNoTemplatePath
	}
	tmpl := template.New("").Funcs(funcs)
	for _, file := range files {
		sub := tmpl.New(file)
		contents, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, fmt.Errorf("error reading %q: %w", file, err)
		}
		_, err = sub.Parse(string(contents))
		if err != nil {
			return nil, fmt.Errorf("error parsing %q: %w", file, err)
		}
	}
	return tmpl, nil
}

// mergeFuncMaps flattens two FuncMaps into one, with the values in `page`
// overriding the values in `in` if they have the same keys.
func mergeFuncMaps(in template.FuncMap, page template.FuncMap) template.FuncMap {
	res := template.FuncMap{}
	for k, v := range in {
		res[k] = v
	}
	for k, v := range page {
		res[k] = v
	}
	return res
}
