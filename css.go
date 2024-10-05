package temple

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
)

// CSSEmbedder is an interface that Components can fulfill to include some CSS
// that should be embedded directly into the rendered HTML. The contents will
// be made available to the template as .EmbeddedCSS.
type CSSEmbedder interface {
	// EmbedCSS returns the CSS, without <style> tags, that should be
	// embedded directly in the output HTML.
	//
	// If this Component embeds any other Components, it should include
	// their EmbedCSS output in its own EmbedCSS output.
	EmbedCSS(context.Context) template.CSS
}

// CSSLinker is an interface that Components can fulfill to include some CSS
// that should be loaded through a <link> element in the template. The contents
// will be made available to the template as .LinkedCSS.
type CSSLinker interface {
	// LinkCSS returns a list of URLs to CSS files that should be linked to
	// from the output HTML.
	//
	// If this Component embeds any other Components, it should include
	// their LinkCSS output in its own LinkCSS output.
	LinkCSS(context.Context) []string
}

func getComponentCSSEmbeds(ctx context.Context, component Component) template.CSS {
	var results template.CSS
	seen := map[string]struct{}{}
	components := getRecursiveComponents(ctx, component)
	for _, comp := range components {
		embed, ok := comp.(CSSEmbedder)
		if !ok {
			continue
		}
		css := embed.EmbedCSS(ctx)
		checksum := hex.EncodeToString(sha256.New().Sum([]byte(css)))
		if _, ok := seen[checksum]; ok {
			continue
		}
		seen[checksum] = struct{}{}
		results += template.CSS(fmt.Sprintf(`
/* embedded CSS from %T */
%s`, comp, css)) // #nosec G203
	}
	return results
}

func getComponentCSSLinks(ctx context.Context, component Component) []string {
	var results []string
	seen := map[string]struct{}{}
	components := getRecursiveComponents(ctx, component)
	for _, comp := range components {
		link, ok := comp.(CSSLinker)
		if !ok {
			continue
		}
		css := link.LinkCSS(ctx)
		for _, source := range css {
			if _, ok := seen[source]; ok {
				continue
			}
			results = append(results, source)
			seen[source] = struct{}{}
		}
	}
	return results
}
