package temple

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// JSEmbedder is an interface that Components can fulfill to include some
// JavaScript that should be embedded directly into the rendered HTML. The
// contents will be made available to the template as .EmbeddedJS.
type JSEmbedder interface {
	// EmbedJS returns the JavaScript, without <script> tags, that should
	// be embedded directly in the output HTML.
	EmbedJS(context.Context) string
}

// JSLinker is an interface that Components can fulfill to include some
// JavaScript that should be loaded separately from the HTML document, using a
// <script> tag with a src attribute. The contents will be made available to
// the template as .LinkedJS.
type JSLinker interface {
	// LinkJS returns a list of URLs to JavaScript files that should be
	// linked to from the output HTML.
	//
	// If this Component embeds any other Components, it should include
	// their LinkJS output in its own LinkJS output.
	LinkJS(context.Context) []string
}

func getComponentJSEmbeds(ctx context.Context, component Component) string {
	var results string
	seen := map[string]struct{}{}
	components := getRecursiveComponents(ctx, component)
	for _, comp := range components {
		embed, ok := comp.(JSEmbedder)
		if !ok {
			continue
		}
		script := embed.EmbedJS(ctx)
		checksum := hex.EncodeToString(sha256.New().Sum([]byte(script)))
		if _, ok := seen[checksum]; ok {
			continue
		}
		seen[checksum] = struct{}{}
		results += fmt.Sprintf(`
/* embedded JavaScript from %T */
%s`, comp, script)
	}
	return results
}

func getComponentJSLinks(ctx context.Context, component Component) []string {
	var results []string
	seen := map[string]struct{}{}
	components := getRecursiveComponents(ctx, component)
	for _, comp := range components {
		link, ok := comp.(JSLinker)
		if !ok {
			continue
		}
		js := link.LinkJS(ctx)
		for _, source := range js {
			if _, ok := seen[source]; ok {
				continue
			}
			results = append(results, source)
			seen[source] = struct{}{}
		}
	}
	return results
}
