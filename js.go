package temple

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"strconv"
	"strings"
)

var (
	// ErrJSInlineTemplatePathNotSet is returned when the TemplatePath
	// property on a JSInline is not set, which isn't valid. The
	// TemplatePath controls what JavaScript to embed, so it should never
	// be omitted.
	ErrJSInlineTemplatePathNotSet = errors.New("JSInline TemplatePath must be set")
)

// jsResource is an abstraction over JSLink and JSInline.
type jsResource interface {
	// getJS returns the JS template string to render for the link or
	// script block.
	getJS(dir fs.FS) (string, error)

	// getKey returns a unique identifier for the template, used when
	// caching it.
	getKey() string

	// equal should return true if the passed jsResource is considered
	// equivalent to the implementing jsResource. This is used when
	// deduplicating resources.
	equal(jsResource) bool
}

// JSRenderData holds the information passed to the template when rendering the
// template for a [JSInline] or [JSLink].
type JSRenderData[SiteType Site, PageType Renderable] struct {
	// JS is the JSInline struct being rendered. It may be empty if this
	// data is for a JSLink instead.
	JS JSInline

	// JSLink is the JSLink struct being rendered. It may be empty if this
	// data is for a JSInline instead.
	JSLink JSLink

	// Site is the caller-defined site type, used for including globals.
	Site SiteType

	// Page is the specific page the JS is being rendered into.
	Page PageType
}

// JSInline holds the necessary information to embed JavaScript directly into a
// page's HTML output, inside a <script> tag.
//
// The TemplatePath should point to a template containing the JavaScript to
// render. The template can use any of the data in the [JSRenderData]. The
// <script> tag should not be included; it will be generated from the rest of
// the properties.
//
// temple may, but is not guaranteed to, merge <script> blocks it identifies as
// mergeable. At the moment, those merge semantics are that all properties
// except the relation calculators are identical, but that logic is subject to
// change. If it is important that a <script> block not be merged into any
// other <script> block, set DisableElementMerge to true.
type JSInline struct {
	// TemplatePath is the path, relative to the Site's TemplateDir, to the
	// template that should be rendered to get the contents of the
	// JavaScript <script> tag. The template should not include the
	// <script> tags. A JSRenderData value will be passed as the template
	// data, with the JSInline property set to this value.
	TemplatePath string

	// CrossOrigin is the value of the crossorigin attribute for the
	// <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#crossorigin
	// for more information.
	CrossOrigin string

	// NoModule indicates whether to include the nomodule attribute for the
	// <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#nomodule
	// for more information.
	NoModule bool

	// Nonce is the value of the nonce attribute for the <script> tag that
	// will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#nonce
	// for more information.
	Nonce string

	// ReferrerPolicy is the value of the referrerpolicy attribute for the
	// <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#referrerpolicy
	// for more information.
	ReferrerPolicy string

	// Type is the value of the type attribute for the <script> tag that
	// will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#type
	// for more information.
	Type string

	// Attrs holds any additional non-standard or unsupported attributes
	// that should be set on the <script> tag that will be generated.
	Attrs map[string]string

	// DisableElementMerge, when set to true, prevents a <script> tag from
	// being merged with any other <script> tag.
	DisableElementMerge bool

	// PlaceInFooter, when set to true, makes this JavaScript part of the
	// FooterJS property of RenderData. Otherwise, it is part of the
	// HeaderJS property of RenderData. This separation exists so some
	// JavaScript can be loaded in the <head> of the document, while other
	// JavaScript can be loaded after the rest of the document has been
	// loaded. Where these properties end up actually being placed is up to
	// the template, but that is the intention.
	PlaceInFooter bool

	// JSInlineRelationCalculator can be used to control how this <script>
	// tag gets rendered in relation to any other <script> tag. If the
	// function returns ResourceRelationshipAfter, this <script> tag will
	// always come after the other <script> tag in the HTML document. If
	// the function returns ResourceRelationshipBefore, this <script> tag
	// will always come before the other <script> tag in the HTML document.
	// If the function returns ResourceRelationshipNeutral, no guarantees
	// are made about where the JavaScript resources will appear relative
	// to each other in the HTML document.
	//
	// If this <script> tag has no requirements about its positioning
	// relative to other JavaScript resources, just let this property be
	// nil.
	JSInlineRelationCalculator func(context.Context, JSInline) ResourceRelationship

	// JSLinkRelationCalculator can be used to control how this <script> tag
	// gets rendered in relation to any other JavaScript <script> tag. If the
	// function returns ResourceRelationshipAfter, this <script> tag will
	// always come after the other <script> tag in the HTML document. If the
	// function returns ResourceRelationshipBefore, this <script> tag will
	// always come before the other <script> tag in the HTML document. If the
	// function returns ResourceRelationshipNeutral, no guarantees are made
	// about where the JavaScript resources will appear relative to each other in
	// the HTML document.
	//
	// These orderings are only guaranteed when comparing resources with
	// the same PlaceInFooter values.
	//
	// If this <script> tag has no requirements about its positioning
	// relative to other JavaScript resources, just let this property be nil.
	JSLinkRelationCalculator func(context.Context, JSLink) ResourceRelationship
}

// getJS returns the string to include in the JavaScript output, using the
// passed fs.FS to load the template path.
func (block JSInline) getJS(dir fs.FS) (string, error) {
	if strings.TrimSpace(block.TemplatePath) == "" {
		return "", ErrJSInlineTemplatePathNotSet
	}
	contents, err := fs.ReadFile(dir, block.TemplatePath)
	if err != nil {
		return "", err
	}
	// html/template doesn't allow setting the type attribute of a script
	// within the template. See https://go.dev/issues/59112. To get around
	// this, we use string concatenation to insert the type ourselves, if
	// necessary.
	typestring := block.Type
	if typestring != "" {
		typestring = ` type=` + strconv.Quote(typestring)
	}
	return `<script` + typestring + `{{if .JS.CrossOrigin }} crossorigin="{{ .JS.CrossOrigin }}"{{ end }}{{ if .JS.NoModule }} nomodule{{ end }}{{ if .JS.Nonce }} nonce="{{.Nonce}}"{{ end }}{{ if .JS.ReferrerPolicy }} referrerpolicy="{{ .JS.ReferrerPolicy }}"{{ end }}{{ range $k, $v := .JS.Attrs }}{{ $k }}{{ if $v }}="{{$v}}"{{ end }}{{ end }}>
` + string(contents) + `
</script>`, nil
}

// getKey returns a cache key for the template for this tag. The cache key
// should be unique to the template literal, without regard to the template
// data.
func (block JSInline) getKey() string {
	return block.TemplatePath
}

// equal returns true if block and other should be considered equal. The
// largest consequence of returning true is that only one will be rendered to
// the page.
func (block JSInline) equal(other jsResource) bool {
	comp, ok := other.(JSInline)
	if !ok {
		return false
	}
	if block.TemplatePath != comp.TemplatePath {
		return false
	}
	if block.CrossOrigin != comp.CrossOrigin {
		return false
	}
	if block.NoModule != comp.NoModule {
		return false
	}
	if block.Nonce != comp.Nonce {
		return false
	}
	if block.ReferrerPolicy != comp.ReferrerPolicy {
		return false
	}
	if block.Type != comp.Type {
		return false
	}
	if !maps.Equal(block.Attrs, comp.Attrs) {
		return false
	}
	if block.DisableElementMerge != comp.DisableElementMerge {
		return false
	}
	if block.PlaceInFooter != comp.PlaceInFooter {
		return false
	}
	return true
}

// JSLink holds the necessary information to include JavaScript in a page's
// HTML output as a <script> with a src attribute that downloads the JavaScript
// file from another URL.
//
// The Src should point to the URL to load the JavaScript file from.
type JSLink struct {
	// Src is the URL to load the JavaScript from. It will be used verbatim
	// as the <script> element's src attribute. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#src
	// for more information.
	Src string

	// Async indicates whether to include the async attribute for the
	// <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#async
	// for more information.
	Async bool

	// AttributionSrc, if set to true, will include the attributionsrc
	// attribute in the <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#attributionsrc
	// for more information.
	AttributionSrc bool

	// AttributionSrcURLs, if non-empty, will set the value of the
	// attributionsrc attribute in the <script> tag that will be generated.
	// See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#attributionsrc
	// for more information.
	//
	// If AttributionSrcURLs is set, AttributionSrc must be set to true, or
	// AttributionSrcURLs will have no effect.
	AttributionSrcURLs string

	// Blocking is the value of the blocking attribute for the <script> tag
	// that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#blocking
	// for more information.
	Blocking string

	// CrossOrigin is the value of the crossorigin attribute for the
	// <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#crossorigin
	// for more information.
	CrossOrigin string

	// Defer indicates whether to include the defer attribute for the
	// <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#defer
	// for more information.
	Defer bool

	// FetchPriority is the value of the fetchpriority attribute for the
	// <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#fetchpriority
	// for more information.
	FetchPriority string

	// Integrity is the value of the integrity attribute for the <script>
	// tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#integrity
	// for more information.
	Integrity string

	// NoModule indicates whether to include the nomodule attribute for the
	// <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#nomodule
	// for more information.
	NoModule bool

	// Nonce is the value of the nonce attribute for the <script> tag that
	// will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#nonce
	// for more information.
	Nonce string

	// ReferrerPolicy is the value of the referrerpolicy attribute for the
	// <script> tag that will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#referrerpolicy
	// for more information.
	ReferrerPolicy string

	// Type is the value of the type attribute for the <script> tag that
	// will be generated. See
	// https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/script#type
	// for more information.
	Type string

	// Attrs holds any additional non-standard or unsupported attributes
	// that should be set on the <script> tag that will be generated.
	Attrs map[string]string

	// TemplatePath is the path, relative to the Site's TemplateDir, to the
	// template that should be rendered to construct the <script> tag from
	// this struct. If left empty, the default template will be used, but
	// it can be specified to override the template if desired. A
	// JSRenderData will be passed to the template with the JSLink property
	// set.
	TemplatePath string

	// PlaceInFooter, when set to true, makes this JavaScript part of the
	// FooterJS property of RenderData. Otherwise, it is part of the
	// HeaderJS property of RenderData. This separation exists so some
	// JavaScript can be loaded in the <head> of the document, while other
	// JavaScript can be loaded after the rest of the document has been
	// loaded. Where these properties end up actually being placed is up to
	// the template, but that is the intention.
	PlaceInFooter bool

	// JSInlineRelationCalculator can be used to control how this <script>
	// tag gets rendered in relation to any other <script> tag. If the
	// function returns ResourceRelationshipAfter, this <script> tag will
	// always come after the other <script> tag in the HTML document. If
	// the function returns ResourceRelationshipBefore, this <script> tag
	// will always come before the other <script> tag in the HTML document.
	// If the function returns ResourceRelationshipNeutral, no guarantees
	// are made about where the JavaScript resources will appear relative
	// to each other in the HTML document.
	//
	// If this <script> tag has no requirements about its positioning
	// relative to other JavaScript resources, just let this property be
	// nil.
	JSInlineRelationCalculator func(context.Context, JSInline) ResourceRelationship

	// JSLinkRelationCalculator can be used to control how this <script> tag
	// gets rendered in relation to any other JavaScript <script> tag. If the
	// function returns ResourceRelationshipAfter, this <script> tag will
	// always come after the other <script> tag in the HTML document. If the
	// function returns ResourceRelationshipBefore, this <script> tag will
	// always come before the other <script> tag in the HTML document. If the
	// function returns ResourceRelationshipNeutral, no guarantees are made
	// about where the JavaScript resources will appear relative to each other in
	// the HTML document.
	//
	// These orderings are only guaranteed when comparing resources with
	// the same PlaceInFooter values.
	//
	// If this <script> tag has no requirements about its positioning
	// relative to other JavaScript resources, just let this property be nil.
	JSLinkRelationCalculator func(context.Context, JSLink) ResourceRelationship
}

// equal returns true if tag and other should be considered equal. The largest
// consequence of returning true is that only one will be rendered to the page.
func (tag JSLink) equal(other jsResource) bool {
	comp, ok := other.(JSLink)
	if !ok {
		return false
	}
	if tag.Src != comp.Src {
		return false
	}
	if tag.Async != comp.Async {
		return false
	}
	if tag.AttributionSrc != comp.AttributionSrc {
		return false
	}
	if tag.AttributionSrcURLs != comp.AttributionSrcURLs {
		return false
	}
	if tag.Blocking != comp.Blocking {
		return false
	}
	if tag.CrossOrigin != comp.CrossOrigin {
		return false
	}
	if tag.Defer != comp.Defer {
		return false
	}
	if tag.FetchPriority != comp.FetchPriority {
		return false
	}
	if tag.Integrity != comp.Integrity {
		return false
	}
	if tag.NoModule != comp.NoModule {
		return false
	}
	if tag.Nonce != comp.Nonce {
		return false
	}
	if tag.ReferrerPolicy != comp.ReferrerPolicy {
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
	if tag.PlaceInFooter != comp.PlaceInFooter {
		return false
	}
	return true
}

// getJS returns the string to include in the JavaScript output, using the
// passed fs.FS to load the template path, if tag.TemplatePath is set.
func (tag JSLink) getJS(dir fs.FS) (string, error) {
	if tag.TemplatePath != "" {
		contents, err := fs.ReadFile(dir, tag.TemplatePath)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}
	// html/template doesn't allow setting the type attribute of a script
	// within the template. See https://go.dev/issues/59112. To get around
	// this, we use string concatenation to insert the type ourselves, if
	// necessary.
	typestring := tag.Type
	if typestring != "" {
		typestring = ` type=` + strconv.Quote(typestring)
	}
	return `<script` + typestring + ` src="{{ .JSLink.Src }}"{{ if .JSLink.Async }} async{{ end }}{{ if .JSLink.AttributionSrc }}attributionsrc{{if .JSLink.AttributionSrcURLs }}="{{ .JSLink.AttributionSrcURLs }}"{{ end }}{{ end }}{{ if .JSLink.Blocking }} blocking="{{ .JSLink.Blocking }}"{{ end }}{{if .JSLink.CrossOrigin }} crossorigin="{{ .JSLink.CrossOrigin }}"{{ end }}{{ if .JSLink.Defer }} defer{{ end }}{{ if .JSLink.FetchPriority }} fetchpriority="{{ .JSLink.FetchPriority }}"{{ end }}{{ if .JSLink.Integrity }} integrity="{{ .JSLink.Integrity }}"{{ end }}{{ if .JSLink.NoModule }} nomodule{{ end }}{{ if .JSLink.Nonce }} nonce="{{.Nonce}}"{{ end }}{{ if .JSLink.ReferrerPolicy }} referrerpolicy="{{ .JSLink.ReferrerPolicy }}"{{ end }}{{ range $k, $v := .JSLink.Attrs }} {{ $k }}{{ if $v }}="{{$v}}"{{ end }}{{ end }}></script>`, nil
}

// getKey returns a cache key for the template for this tag. The cache key
// should be unique to the template literal, without regard to the template
// data.
func (tag JSLink) getKey() string {
	if tag.TemplatePath != "" {
		return tag.TemplatePath
	}
	return ":::impractical.co/temple:defaultJSLinkTemplate"
}

// JSEmbedder is an interface that Components can fulfill to include some
// JavaScript that should be embedded directly into the rendered HTML. The
// contents will be made available to the template as .HeaderJS or .FooterJS,
// depending on whether their PlaceInFooter property is set to true or not.
type JSEmbedder interface {
	// EmbedJS returns the JavaScript, without <script> tags, that should
	// be embedded directly in the output HTML.
	EmbedJS(context.Context) []JSInline
}

// JSLinker is an interface that Components can fulfill to include some
// JavaScript that should be loaded separately from the HTML document, using a
// <script> tag with a src attribute. The contents will be made available to
// the template as .HeaderJS or .FooterJS, depending on whether their
// PlaceInFooter property is set to true or not.
type JSLinker interface {
	// LinkJS returns a list of URLs to JavaScript files that should be
	// linked to from the output HTML.
	//
	// If this Component embeds any other Components, it should include
	// their LinkJS output in its own LinkJS output.
	LinkJS(context.Context) []JSLink
}
