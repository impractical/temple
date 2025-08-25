package temple

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"strings"
)

var (
	// ErrCSSInlineTemplatePathNotSet is returned when the TemplatePath
	// property on a CSSInline is not set, which isn't valid. The
	// TemplatePath controls what CSS to embed, so it should never be
	// omitted.
	ErrCSSInlineTemplatePathNotSet = errors.New("CSSInline TemplatePath must be set")
)

// cssResource is an abstraction over CSSLink and CSSInline.
type cssResource interface {
	// getCSS returns the CSS template string to render for the link or
	// style block.
	getCSS(dir fs.FS) (string, error)

	// getKey returns a unique identifier for the template, used when
	// caching it.
	getKey() string

	// equal should return true if the passed cssResource is considered
	// equivalent to the implementing cssResource. This is used when
	// deduplicating resources.
	equal(cssResource) bool
}

// CSSRenderData holds the information passed to the template when rendering
// the template for a [CSSInline] or [CSSLink].
type CSSRenderData[SiteType Site, PageType Page] struct {
	// CSS is the CSSInline struct being rendered. It may be empty if this
	// data is for a CSSLink instead.
	CSS CSSInline

	// CSSLink is the CSSLink struct being rendered. It may be empty if
	// this data is for a CSSInline instead.
	CSSLink CSSLink

	// Site is the caller-defined site type, used for including globals.
	Site SiteType

	// Page is the specific page the CSS is being rendered into.
	Page PageType
}

// CSSInline holds the necessary information to embed CSS directly into a
// page's HTML output, inside a <style> tag.
//
// The TemplatePath should point to a template containing the CSS to render.
// The template can use any of the data in the [CSSRenderData]. The <style> tag
// should not be included; it will be generated from the rest of the
// properties.
//
// temple may, but is not guaranteed to, merge <style> blocks it identifies as
// mergeable. At the moment, those merge semantics are that all properties
// except the relation calculators are identical, but that logic is subject to
// change. If it is important that a <style> block not be merged into any other
// <style> block, set DisableElementMerge to true.
type CSSInline struct {
	// TemplatePath is the path, relative to the Site's TemplateDir, to the
	// template that should be rendered to get the contents of the CSS
	// <style> block. The template should not include the <style> tags. A
	// CSSRenderData value will be passed as the template data, with the
	// CSSInline property set to this value.
	TemplatePath string

	// Blocking is the value of the blocking attribute for the <style> tag
	// that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/style#blocking
	// for more information.
	Blocking string

	// Media is the value of the media attribute for the <style> tag that
	// will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/style#media
	// for more information.
	Media string

	// Nonce is the value of the nonce attribute for the <style> tag that
	// will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/style#nonce
	// for more information.
	Nonce string

	// Title is the value of the title attribute for the <style> tag that
	// will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/style#title
	// for more information.
	Title string

	// Attrs holds any additional non-standard or unsupported attributes
	// that should be set on the <style> tag that will be generated.
	Attrs map[string]string

	// DisableElementMerge, when set to true, prevents a <style> block from
	// being merged with any other <style> block.
	DisableElementMerge bool

	// DisableImplicitOrdering, when set to true, disables the implicit
	// ordering of resources within a Component for this block. It will not
	// be required to come after the block before it in the []CSSInline,
	// and the block after it will not be required to be rendered after it.
	DisableImplicitOrdering bool

	// CSSLinkRelationCalculator can be used to control how this <link> tag
	// gets rendered in relation to any other CSS <link> tag. If the
	// function returns ResourceRelationshipAfter, this <link> tag will
	// always come after the other <link> tag in the HTML document. If the
	// function returns ResourceRelationshipBefore, this <link> tag will
	// always come before the other <link> tag in the HTML document. If the
	// function returns ResourceRelationshipNeutral, no guarantees are made
	// about where the CSS resources will appear relative to each other in
	// the HTML document.
	//
	// If this <link> tag has no requirements about its positioning
	// relative to other CSS resources, just let this property be nil.
	CSSLinkRelationCalculator func(context.Context, CSSLink) ResourceRelationship

	// CSSInlineRelationCalculator can be used to control how this <style>
	// block gets rendered in relation to any other <style> block. If the
	// function returns ResourceRelationshipAfter, this <style> block will
	// always come after the other <style> block in the HTML document. If
	// the function returns ResourceRelationshipBefore, this <style> block
	// will always come before the other <style> block in the HTML
	// document. If the function returns ResourceRelationshipNeutral, no
	// guarantees are made about where the CSS resources will appear
	// relative to each other in the HTML document.
	//
	// If this <style> block has no requirements about its positioning
	// relative to other CSS resources, just let this property be nil.
	CSSInlineRelationCalculator func(context.Context, CSSInline) ResourceRelationship
}

// equal returns true if block and other should be considered equal. The
// largest consequence of returning true is that only one will be rendered to
// the page.
func (block CSSInline) equal(other cssResource) bool {
	comp, ok := other.(CSSInline)
	if !ok {
		return false
	}
	if block.TemplatePath != comp.TemplatePath {
		return false
	}
	if block.Blocking != comp.Blocking {
		return false
	}
	if block.Media != comp.Media {
		return false
	}
	if block.Nonce != comp.Nonce {
		return false
	}
	if block.Title != comp.Title {
		return false
	}
	if !maps.Equal(block.Attrs, comp.Attrs) {
		return false
	}
	if block.DisableElementMerge != comp.DisableElementMerge {
		return false
	}
	return true
}

// getCSS returns the string to include in the CSS output, using the passed
// fs.FS to load the template path.
func (block CSSInline) getCSS(dir fs.FS) (string, error) {
	if strings.TrimSpace(block.TemplatePath) == "" {
		return "", ErrCSSInlineTemplatePathNotSet
	}
	contents, err := fs.ReadFile(dir, block.TemplatePath)
	if err != nil {
		return "", err
	}
	return `<style{{ if .CSS.Blocking }} blocking="{{ .CSS.Blocking }}"{{ end }}{{ if .CSS.Media }} media="{{ .CSS.Media }}"{{ end }}{{ if .CSS.Nonce }} nonce="{{ .CSS.Nonce }}"{{ end }}{{ if .CSS.Title }} title="{{ .CSS.Title }}"{{ end }}{{ range $key, $val := .CSS.Attrs }} {{ $key }}="{{ $val }}"{{ end }}>
` + string(contents) + `
</style>`, nil
}

// getKey returns a cache key for the template for this tag. The cache key
// should be unique to the template literal, without regard to the template
// data.
func (block CSSInline) getKey() string {
	return block.TemplatePath
}

// CSSLink holds the necessary information to include CSS in a page's HTML
// output as a <link>, which downloads the CSS file from another URL.
//
// The Href should point to the URL to load the CSS file from.
type CSSLink struct {
	// Href is the URL to load the CSS from. It will be used verbatim as
	// the <link> element's href attribute. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#href
	// for more information.
	Href string

	// Rel is the value of the rel attribute for the <link> tag that will be
	// generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Attributes/rel
	// for more information.
	Rel string

	// As is the value of the as attribute for the <link> tag that will be
	// generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#as
	// for more information.
	As string

	// Blocking is the value of the blocking attribute for the <link> tag
	// that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#blocking
	// for more information.
	Blocking string

	// CrossOrigin is the value of the crossorigin attribute for the <link>
	// tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Attributes/crossorigin
	// for more information.
	CrossOrigin string

	// Disabled indicates whether to include the disabled attribute in the
	// <link> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#disabled
	// for more information.
	Disabled bool

	// FetchPriority is the value of the fetchpriority attribute for the
	// <link> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#fetchpriority
	// for more information.
	FetchPriority string

	// Integrity is the value of the integrity attribute for the <link> tag
	// that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#integrity
	// for more information.
	Integrity string

	// Media is the value of the media attribute for the <link> tag that
	// will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#media
	// for more information.
	Media string

	// ReferrerPolicy is the value of the referrerpolicy attribute for the
	// <link> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#referrerpolicy
	// for more information.
	ReferrerPolicy string

	// Title is the value of the title attribute for the <link> tag that
	// will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#title
	// for more information.
	Title string

	// Type is the value of the type attribute for the <link> tag that will
	// be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/link#type
	// for more information.
	Type string

	// Attrs holds any additional non-standard or unsupported attributes
	// that should be set on the <link> tag that will be generated.
	Attrs map[string]string

	// DisableImplicitOrdering, when set to true, disables the implicit
	// ordering of resources within a Component for this link. It will not
	// be required to come after the link before it in the []CSSLink, and
	// the link after it will not be required to be rendered after it.
	DisableImplicitOrdering bool

	// CSSLinkRelationCalculator can be used to control how this <link> tag
	// gets rendered in relation to any other CSS link tag. If the function
	// returns ResourceRelationshipAfter, this <link> tag will always come
	// after the other link in the HTML document. If the function returns
	// ResourceRelationshipBefore, this <link> tag will always come before
	// the other link in the HTML document. If the function returns
	// ResourceRelationshipNeutral, no guarantees are made about where the
	// CSS resources will appear relative to each other in the HTML
	// document.
	//
	// If this <link> tag has no requirements about its positioning
	// relative to other CSS resources, just let this property be nil.
	CSSLinkRelationCalculator func(context.Context, CSSLink) ResourceRelationship

	// CSSInlineRelationCalculator can be used to control how this <link>
	// tag gets rendered in relation to any <style> block. If the function
	// returns ResourceRelationshipAfter, this <link> tag will always come
	// after the <style> block in the HTML document. If the function
	// returns ResourceRelationshipBefore, this <link> tag will always come
	// before the <style> block in the HTML document. If the function
	// returns ResourceRelationshipNeutral, no guarantees are made about
	// where the CSS resources will appear relative to each other in the
	// HTML document.
	//
	// If this <link> tag has no requirements about its positioning
	// relative to other CSS resources, just let this property be nil.
	CSSInlineRelationCalculator func(context.Context, CSSInline) ResourceRelationship

	// TemplatePath is the path, relative to the Site's TemplateDir, to the
	// template that should be rendered to construct the <link> tag from
	// this struct. If left empty, the default template will be used, but
	// it can be specified to override the template if desired. A
	// CSSRenderData will be passed to the template with the CSSLink
	// property set.
	TemplatePath string
}

// equal returns true if tag and other should be considered equal. The largest
// consequence of returning true is that only one will be rendered to the page.
func (tag CSSLink) equal(other cssResource) bool {
	comp, ok := other.(CSSLink)
	if !ok {
		return false
	}
	if tag.Href != comp.Href {
		return false
	}
	if tag.Rel != comp.Rel {
		return false
	}
	if tag.As != comp.As {
		return false
	}
	if tag.Blocking != comp.Blocking {
		return false
	}
	if tag.CrossOrigin != comp.CrossOrigin {
		return false
	}
	if tag.Disabled != comp.Disabled {
		return false
	}
	if tag.FetchPriority != comp.FetchPriority {
		return false
	}
	if tag.Integrity != comp.Integrity {
		return false
	}
	if tag.Media != comp.Media {
		return false
	}
	if tag.ReferrerPolicy != comp.ReferrerPolicy {
		return false
	}
	if tag.Title != comp.Title {
		return false
	}
	if tag.Type != comp.Type {
		return false
	}
	if !maps.Equal(tag.Attrs, comp.Attrs) {
		return false
	}
	if tag.TemplatePath != comp.TemplatePath {
		return false
	}
	return true
}

// getCSS returns the string to include in the CSS output, using the passed
// fs.FS to load the template path, if tag.TemplatePath is set.
func (tag CSSLink) getCSS(dir fs.FS) (string, error) {
	if tag.TemplatePath != "" {
		contents, err := fs.ReadFile(dir, tag.TemplatePath)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}
	return `<link{{ if .CSSLink.Href}} href="{{ .CSSLink.Href }}"{{ end }}{{ if .CSSLink.Rel }} rel="{{ .CSSLink.Rel }}"{{ end }}{{ if .CSSLink.As }} as="{{ .CSSLink.As }}"{{ end }}{{ if .CSSLink.Blocking }} blocking="{{ .CSSLink.Blocking }}"{{ end }}{{ if .CSSLink.CrossOrigin }} crossorigin="{{ .CSSLink.CrossOrigin }}"{{ end }}{{ if .CSSLink.Disabled }} disabled{{ end }}{{ if .CSSLink.FetchPriority }} fetchpriority="{{ .CSSLink.FetchPriority }}"{{ end }}{{ if .CSSLink.Integrity }} integrity="{{ .CSSLink.Integrity }}"{{ end }}{{ if .CSSLink.Media }}media="{{ .CSSLink.Media }}"{{ end }}{{ if .CSSLink.ReferrerPolicy }} referrerpolicy="{{ .CSSLink.ReferrerPolicy }}"{{ end }}{{ if .CSSLink.Title }} title="{{ .CSSLink.Title }}"{{ end }}{{ if .CSSLink.Type }} type="{{ .CSSLink.Type }}"{{ end }}{{ range $key, $val := .CSSLink.Attrs }} {{ $key }}="{{ $val }}"{{ end }}>`, nil
}

// getKey returns a cache key for the template for this tag. The cache key
// should be unique to the template literal, without regard to the template
// data.
func (tag CSSLink) getKey() string {
	if tag.TemplatePath != "" {
		return tag.TemplatePath
	}
	return ":::impractical.co/temple:defaultCSSLinkTemplate"
}

// CSSEmbedder is an interface that Components can fulfill to include some CSS
// that should be embedded directly into the rendered HTML. The contents will
// be made available to the template as .CSS.
type CSSEmbedder interface {
	// EmbedCSS returns the CSSInline values that describe the CSS to embed
	// directly in the output HTML.
	EmbedCSS(context.Context) []CSSInline
}

// CSSLinker is an interface that Components can fulfill to include some CSS
// that should be loaded through a <link> element in the template. The contents
// will be made available to the template as .CSS.
type CSSLinker interface {
	// LinkCSS returns a list of CSSLink values that describe the CSS links
	// to include in the output HTML.
	LinkCSS(context.Context) []CSSLink
}
