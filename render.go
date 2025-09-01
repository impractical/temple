package temple

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"maps"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	// Templates returns a list of filepaths to html/template contents that
	// need to be parsed before the component can be rendered.
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

// Page is an interface for a page that can be passed to Render. It defines a
// single logical page of the application, composed of one or more Components.
// It should contain all the information needed to render the Components to
// HTML.
type Page interface {
	Component

	// Key is a unique key to use when caching this page so it doesn't need
	// to be re-parsed. A good key is consistent, but unique per Page.
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
type RenderData[SiteType Site, PageType Page] struct {
	// Site is an instance of the Site type, containing all the
	// configuration and information about a Site. This can be used to
	// avoid passing global configuration options to every single page.
	Site SiteType

	// Page is the information for a specific page, embedded in that page's
	// Page type.
	Page PageType

	// CSS contains any <style> or <link> tags containing CSS that need to
	// be rendered. It should be rendered as part of the page's <head> tag.
	CSS template.HTML

	// HeaderJS contains any <script> tags intended to be rendered at the
	// head of the document, before any displayed elements have been
	// loaded. It is usually rendered in the page's <head> tag.
	HeaderJS template.HTML

	// FooterJS contains any <script> tags intended to be rendered at the
	// end of a document, after any displayed elements have been loaded. It
	// is usually rendered right before the page's closing </body> tag.
	FooterJS template.HTML
}

// RenderOption is a private interface that modifies how the [Render] function
// behaves. It's returned by a [RenderConfigurer].
type RenderOption interface {
	setRenderOpts(*renderOpts)
}

type renderOpts struct {
	disablePageBuffering bool
}

// RenderConfigurer is an interface that a [Site], [Page], or [Component] can
// fill that will modify how [Render] behaves when that Site, Page, or
// Component is being rendered. In case more than one [Site], [Page], or
// [Component] implements [RenderConfigurer], the [Component] will beat the
// [Page] in any conflict, and the [Page] will beat the [Site]. Multiple
// Components with conflicting [RenderOption]s have undefined behavior and the
// winner is not guaranteed.
type RenderConfigurer interface {
	// ConfigureRender returns a list of options that modify how the Render
	// function behaves.
	ConfigureRender() []RenderOption
}

type renderOptionFunc func(*renderOpts)

func (function renderOptionFunc) setRenderOpts(in *renderOpts) {
	function(in)
}

// RenderOptionDisablePageBuffering provides a [RenderOption] for controlling
// whether the rendered HTML will be buffered in memory before being written to
// the [http.ResponseWriter] or not. The default is to buffer the HTML so any
// errors in rendering it can be caught before the first byte is written. To
// disable this default, return this function as a [RenderOption] from a
// [RenderConfigurer], passing `true`. To override a previous [RenderOption]
// and turn the buffering back on, pass `false`.
func RenderOptionDisablePageBuffering(disable bool) RenderOption { //nolint:ireturn // have to return an interface, that's the whole point
	return renderOptionFunc(func(opts *renderOpts) {
		opts.disablePageBuffering = disable
	})
}

// Render renders the passed Page to the Writer. If it can't, a server error
// page is written instead. If the Site implements ServerErrorPager, that will
// be rendered; if not, a simple text page indicating a server error will be
// written.
func Render[SiteType Site, PageType Page](ctx context.Context, out io.Writer, site SiteType, page PageType) {
	defer func() {
		// if the ResponseWriter can be closed, let's try to close it
		if closer, ok := out.(io.Closer); ok {
			err := closer.Close()
			// if there's an error closing it, logging it's about all we can do
			if err != nil {
				logger(ctx).
					ErrorContext(ctx, "error closing response writer", "error", err)
			}
		}
	}()

	tracer := otel.GetTracerProvider().Tracer("impractical.co/temple")
	var span trace.Span
	ctx, span = tracer.Start(ctx, "render")
	defer span.End()

	opts := renderOpts{}

	if renderConfigurer, ok := any(site).(RenderConfigurer); ok {
		for _, opt := range renderConfigurer.ConfigureRender() {
			opt.setRenderOpts(&opts)
		}
	}

	if renderConfigurer, ok := any(page).(RenderConfigurer); ok {
		for _, opt := range renderConfigurer.ConfigureRender() {
			opt.setRenderOpts(&opts)
		}
	}

	components := getRecursiveComponents(ctx, page)
	for _, component := range components {
		if renderConfigurer, ok := any(component).(RenderConfigurer); ok {
			for _, opt := range renderConfigurer.ConfigureRender() {
				opt.setRenderOpts(&opts)
			}
		}
	}

	// try to render the page
	err := basicRender(ctx, out, site, page, components, opts)

	// if there's no error, we're done here
	if err == nil {
		return
	}

	// if there is an error, we now need to try and render a server error
	// page

	// but first we're logging whatever went wrong
	logger(ctx).
		ErrorContext(ctx, "error rendering page", "error", err)

	span.AddEvent("error rendering page",
		trace.WithStackTrace(true),
		trace.WithAttributes(attribute.String("error", err.Error())),
	)

	// now let's render the server error page
	if pager, ok := Site(site).(ServerErrorPager); ok {
		errPage := pager.ServerErrorPage(ctx)
		components := getRecursiveComponents(ctx, errPage)
		err = basicRender(ctx, out, site, pager.ServerErrorPage(ctx), components, opts)
		if err != nil {
			// if we can't do that, everything's doomed, doomed, doomed
			// just log it and we'll move on
			logger(ctx).
				ErrorContext(ctx, "error rendering server error page", "error", err)
		}
		return
	}

	// there's no default server error page, write a server error message
	_, err = out.Write([]byte("Server error."))
	if err != nil {
		logger(ctx).
			ErrorContext(ctx, "error writing server error message", "error", err)
	}
}

func basicRender[SiteType Site, PageType Page](ctx context.Context, output io.Writer, site SiteType, page PageType, components []Component, opts renderOpts) error { //nolint:revive // yeah six is a lot of args, but them's the breaks
	tmpl, err := getTemplate(ctx, site, page)
	if err != nil {
		return err
	}

	graphs := buildGraphs(ctx, components)
	cssResources, err := walkGraph(ctx, graphs.css)
	if err != nil {
		return err
	}
	headJSResources, err := walkGraph(ctx, graphs.headJS)
	if err != nil {
		return err
	}
	footJSResources, err := walkGraph(ctx, graphs.footJS)
	if err != nil {
		return err
	}
	var css, headJS, footJS strings.Builder
	cssTmpl := template.New("")
	headJSTmpl := template.New("")
	footJSTmpl := template.New("")
	for _, cssResource := range cssResources {
		err = parseResource(ctx, site, cssResource.getCSS, cssResource.getKey(), cssTmpl)
		if err != nil {
			return err
		}
	}
	for _, jsResource := range headJSResources {
		err = parseResource(ctx, site, jsResource.getJS, jsResource.getKey(), headJSTmpl)
		if err != nil {
			return err
		}
	}
	for _, jsResource := range footJSResources {
		err = parseResource(ctx, site, jsResource.getJS, jsResource.getKey(), footJSTmpl)
		if err != nil {
			return err
		}
	}
	// loop through again, now that everything has been parsed
	for _, cssResource := range cssResources {
		key := cssResource.getKey()
		data := CSSRenderData[SiteType, PageType]{
			Site: site,
			Page: page,
		}
		if inline, ok := cssResource.(CSSInline); ok {
			data.CSS = inline
		}
		if link, ok := cssResource.(CSSLink); ok {
			data.CSSLink = link
		}
		// TODO: combine CSS style blocks, if possible
		err = cssTmpl.ExecuteTemplate(&css, key, data)
		if err != nil {
			return fmt.Errorf("error executing CSS template %q for %T: %w", key, page, err)
		}
		_, err = css.WriteString("\n")
		if err != nil {
			return err
		}
	}
	for _, jsResource := range footJSResources {
		key := jsResource.getKey()
		data := JSRenderData[SiteType, PageType]{
			Site: site,
			Page: page,
		}
		if inline, ok := jsResource.(JSInline); ok {
			data.JS = inline
		}
		if link, ok := jsResource.(JSLink); ok {
			data.JSLink = link
		}
		// TODO: combine JS <script> blocks, if possible
		err = footJSTmpl.ExecuteTemplate(&footJS, key, data)
		if err != nil {
			return fmt.Errorf("error executing foot JS template %q for %T: %w", key, page, err)
		}
		_, err = footJS.WriteString("\n")
		if err != nil {
			return err
		}
	}
	for _, jsResource := range headJSResources {
		key := jsResource.getKey()
		data := JSRenderData[SiteType, PageType]{
			Site: site,
			Page: page,
		}
		if inline, ok := jsResource.(JSInline); ok {
			data.JS = inline
		}
		if link, ok := jsResource.(JSLink); ok {
			data.JSLink = link
		}
		// TODO: combine JS <script> blocks, if possible
		err = headJSTmpl.ExecuteTemplate(&headJS, key, data)
		if err != nil {
			return fmt.Errorf("error executing head JS template %q for %T: %w", key, page, err)
		}
		_, err = headJS.WriteString("\n")
		if err != nil {
			return err
		}
	}

	data := RenderData[SiteType, PageType]{
		Site:     site,
		Page:     page,
		CSS:      template.HTML(css.String()),    //nolint:gosec // we trust this HTML, people should not let attackers define arbitrary CSS/JS
		HeaderJS: template.HTML(headJS.String()), //nolint:gosec // we trust this HTML, people should not let attackers define arbitrary CSS/JS
		FooterJS: template.HTML(footJS.String()), //nolint:gosec // we trust this HTML, people should not let attackers define arbitrary CSS/JS
	}

	executed := page.ExecutedTemplate(ctx)
	writer := output
	var bufferedWriter bytes.Buffer
	if !opts.disablePageBuffering {
		writer = &bufferedWriter
	}
	err = tmpl.ExecuteTemplate(writer, executed, data)
	if err != nil {
		return fmt.Errorf("error executing template %q for %T: %w", executed, page, err)
	}
	if !opts.disablePageBuffering {
		_, err = io.Copy(output, &bufferedWriter)
		if err != nil {
			return fmt.Errorf("error copying buffered output: %w", err)
		}
	}
	return nil
}

func parseResource(ctx context.Context, site Site, getFunc func(fs.FS) (string, error), key string, target *template.Template) error { //nolint:revive // yeah, 5 args is a lot, but I can't see any way to fix this one
	span := trace.SpanFromContext(ctx)
	var body string
	if cache, ok := site.(ResourceCacher); ok {
		cached := cache.GetCachedResource(ctx, key)
		if cached != nil {
			span.AddEvent("got cached resource",
				trace.WithAttributes(attribute.String("key", key),
					attribute.String("body", *cached)),
			)
			body = *cached
		}
	}
	if body == "" {
		read, err := getFunc(site.TemplateDir(ctx))
		if err != nil {
			return err
		}
		span.AddEvent("read uncached resource from fs",
			trace.WithAttributes(attribute.String("key", key),
				attribute.String("body", read)),
		)
		body = read
	}
	if cache, ok := site.(ResourceCacher); ok {
		cache.SetCachedResource(ctx, key, body)
	}
	_, err := target.New(key).Parse(body)
	if err != nil {
		return err
	}
	span.AddEvent("parsed resource",
		trace.WithAttributes(attribute.String("key", key)),
	)
	return nil
}

func getTemplate(ctx context.Context, site Site, page Page) (*template.Template, error) {
	span := trace.SpanFromContext(ctx)
	key := page.Key(ctx)
	tmplPaths := getComponentTemplatePaths(ctx, page)
	if len(tmplPaths) < 1 {
		return nil, fmt.Errorf("error rendering %T: %w", page, ErrNoTemplatePath)
	}
	if cache, ok := site.(TemplateCacher); ok {
		cached := cache.GetCachedTemplate(ctx, key)
		if cached != nil {
			span.AddEvent("got cached template",
				trace.WithAttributes(attribute.String("key", key)),
			)
			neededTemplates := make(map[string]struct{}, len(tmplPaths))
			for _, path := range tmplPaths {
				neededTemplates[path] = struct{}{}
			}
			for _, cachedTmpl := range cached.Templates() {
				if _, ok := neededTemplates[cachedTmpl.Name()]; !ok {
					continue
				}
				delete(neededTemplates, cachedTmpl.Name())
			}
			if len(neededTemplates) < 1 {
				return cached, nil
			}
			missingTemplates := make([]string, 0, len(neededTemplates))
			for path := range neededTemplates {
				missingTemplates = append(missingTemplates, path)
			}
			span.AddEvent("templates expected and templates in the parse tree didn't match, ignoring cached template",
				trace.WithAttributes(attribute.StringSlice("missing_templates", missingTemplates)))
		}
	}
	funcMap := getComponentFuncMap(ctx, site, page)
	parsed, err := parseTemplates(site.TemplateDir(ctx), funcMap, tmplPaths...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates %v for page %T: %w", tmplPaths, page, err)
	}
	if cache, ok := site.(TemplateCacher); ok {
		cache.SetCachedTemplate(ctx, key, parsed)
	}
	span.AddEvent("parsed templates",
		trace.WithAttributes(attribute.String("key", key)),
		trace.WithAttributes(attribute.StringSlice("templates", tmplPaths)),
	)
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
	maps.Copy(res, in)
	maps.Copy(res, page)
	return res
}
