package temple

import (
	"context"
	"html/template"
	"io/fs"
	"sync"
)

// Site is an interface for the singleton that will be used to render HTML.
// Consumers should use it to store any clients or cross-request state they
// need, and use it to render Pages.
//
// A Site needs to be able to surface the templates it relies on as an fs.FS.
type Site interface {
	// TemplateDir returns an fs.FS containing all the templates needed to
	// render every Page on the Site.
	//
	// The path to templates within the fs.FS should match the output of
	// TemplatePaths for Components.
	TemplateDir(ctx context.Context) fs.FS
}

// TemplateCacher is an optional interface for Sites. Those fulfilling it can
// cache their template parsing using the output of Key from each Page to save
// on the overhead of parsing the template each time. The templates being
// parsed for a given key should be the same every time, as should the template
// getting executed, but the data may still be different, so the output HTML
// cannot be safely presumed to be cacheable.
type TemplateCacher interface {
	// GetCachedTemplate returns the *template.Template specified by the
	// passed key. It should return nil if the template hasn't been cached
	// yet.
	GetCachedTemplate(ctx context.Context, key string) *template.Template

	// SetCachedTemplate stores the passed *template.Template under the
	// passed key, for later retrieval with GetCachedTemplate.
	//
	// Any errors encountered should be logged, but as this is a
	// best-effort operation, will not be surfaced outside the function.
	SetCachedTemplate(ctx context.Context, key string, tmpl *template.Template)
}

// ResourceCacher is an optional interface for Sites. Those fulfilling it can
// cache the templates their resources use, to save on the overhead of parsing
// the template each time. The templates being parsed for a given key should be
// the same every time, as should the template getting executed, but the data
// may still be different, so the output HTML cannot be safely presumed to be
// cacheable.
type ResourceCacher interface {
	// GetCachedResource returns the *string specified by the passed key.
	// It should return nil if the template hasn't been cached yet.
	GetCachedResource(ctx context.Context, key string) *string

	// SetCachedResource stores the passed string under the passed key, for
	// later retrieval with GetCachedResource.
	//
	// Any errors encountered should be logged, but as this is a
	// best-effort operation, will not be surfaced outside the function.
	SetCachedResource(ctx context.Context, key, resource string)
}

// ServerErrorPager defines an interface that Sites can optionally implement.
// If a Site implements ServerErrorPager and Render encounters an error,
// the output of ServerErrorPage will be rendered.
type ServerErrorPager interface {
	ServerErrorPage(ctx context.Context) Page
}

var _ Site = &CachedSite{}
var _ TemplateCacher = &CachedSite{}
var _ ResourceCacher = &CachedSite{}

// CachedSite is an implementation of the Site interface that can be embedded
// in other Site implementations. It fulfills the Site interface and the
// TemplateCacher interface, caching templates in memory and exposing the
// template fs.FS passed to it in NewCachedSite. A CachedSite must be
// instantiated through NewCachedSite, its empty value is not usable.
type CachedSite struct {
	// cache our templates to avoid re-parsing them for every request
	// but allow us to assign funcmaps to them from the page
	templateCache   map[string]*template.Template
	templateCacheMu sync.RWMutex

	resourceCache   map[string]string
	resourceCacheMu sync.RWMutex

	// templateDir is where Render will look for the templates required by
	// Components.
	templateDir fs.FS
}

// NewCachedSite returns a CachedSite instance that is ready to be used.
func NewCachedSite(templates fs.FS) *CachedSite {
	return &CachedSite{
		templateCache: map[string]*template.Template{},
		templateDir:   templates,
		resourceCache: map[string]string{},
	}
}

// GetCachedTemplate returns the cached template associated with the passed
// key, if one exists. If no template is cached for that key, it returns nil.
//
// It can safely be used by multiple goroutines.
func (s *CachedSite) GetCachedTemplate(_ context.Context, key string) *template.Template {
	s.templateCacheMu.RLock()
	defer s.templateCacheMu.RUnlock()
	res, ok := s.templateCache[key]
	if !ok {
		return nil
	}
	return res
}

// SetCachedTemplate caches a template for the given key.
//
// It can safely be used by multiple goroutines.
func (s *CachedSite) SetCachedTemplate(_ context.Context, key string, tmpl *template.Template) {
	s.templateCacheMu.Lock()
	defer s.templateCacheMu.Unlock()
	s.templateCache[key] = tmpl
}

// GetCachedResource returns the cached resource associated with the passed
// key, if one exists. If no resource is cached for that key, it returns nil.
//
// It can safely be used by multiple goroutines.
func (s *CachedSite) GetCachedResource(_ context.Context, key string) *string {
	s.resourceCacheMu.RLock()
	defer s.resourceCacheMu.RUnlock()
	res, ok := s.resourceCache[key]
	if !ok {
		return nil
	}
	return &res
}

// SetCachedResource caches a resource for the given key.
//
// It can safely be used by multiple goroutines.
func (s *CachedSite) SetCachedResource(_ context.Context, key, resource string) {
	s.resourceCacheMu.Lock()
	defer s.resourceCacheMu.Unlock()
	s.resourceCache[key] = resource
}

// TemplateDir returns an fs.FS containing all the templates needed to render a
// Site's Components. In this case, we just pass back what the consumer passed
// in.
func (s *CachedSite) TemplateDir(_ context.Context) fs.FS {
	return s.templateDir
}
