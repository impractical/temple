package temple

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

var (
	// ErrResourceCycle is returned when a dependency cycle between
	// resources is found. This should never happen; it means that the
	// resource another resource depends on itself depends on that other
	// resource. It always indicates a misconfiguration of the resource
	// dependency graph, and means that the ResourceRelationship returned
	// from the RelationCalculator property on a struct is problematic.
	ErrResourceCycle = errors.New("resource cycle detected")
)

// graph is a directed acyclic graph of type Type. It's used to ensure ordering
// constraints of CSS and JS assets are met.
type graph[Type any] struct {
	// nodes holds the nodes in the graph.
	nodes []Type

	// edgesTo holds graph edges, with the key being the position of the
	// node in the nodes slice that the edges are pointing to. It is a list
	// of edges indexed by what they're pointing to.
	//
	// if there's a node 1 and a node 2, and an edge from 1->2, edgesTo
	// will have a key of 2 with a value of [1].
	//
	// nodes point to their dependencies and dependencies are always
	// walked first; i.e., if there's a node 1 and a node 2, and an edge
	// from 1->2, 2 will always appear before 1 when walking the graph.
	edgesTo map[int]map[int]struct{}

	// edgesFrom holds graph edges, with the key being the position of the
	// node in the nodes slice that the edges are pointing from. It is a
	// list of edges indexed by what's doing the pointing.
	//
	// if there's a node 1 and a node 2, and an edge from 1->2, edgesFrom
	// will have a key of 1 with a value of [2].
	//
	// nodes point to their dependencies and dependencies are always
	// walked first; i.e., if there's a node 1 and a node 2, and an edge
	// from 1->2, 2 will always appear before 1 when walking the graph.
	edgesFrom map[int]map[int]struct{}
}

// resourceGraphs is a collection of graphs, one for CSS resources, one for
// JavaScript resources that should be included in the page header, and one for
// JavaScript resources that should be included in the page footer.
type resourceGraphs struct {
	css    graph[cssResource]
	headJS graph[jsResource]
	footJS graph[jsResource]
}

// buildGraphs creates a resourceGraphs containing all the resources that the
// passed components define, with all their dependencies computed.
//
// Each component's resources will have an implicit dependency on the previous
// resource of their type for that component, so their order within the slice
// will be preserved when rendering them.
func buildGraphs(ctx context.Context, components []Component) resourceGraphs {
	result := resourceGraphs{
		css: graph[cssResource]{
			edgesTo:   map[int]map[int]struct{}{},
			edgesFrom: map[int]map[int]struct{}{},
		},
		headJS: graph[jsResource]{
			edgesTo:   map[int]map[int]struct{}{},
			edgesFrom: map[int]map[int]struct{}{},
		},
		footJS: graph[jsResource]{
			edgesTo:   map[int]map[int]struct{}{},
			edgesFrom: map[int]map[int]struct{}{},
		},
	}
	for _, component := range components {
		if cssLinker, ok := component.(CSSLinker); ok {
			links := cssLinker.LinkCSS(ctx)
			lastLink := -1
			for _, link := range links {
				if slices.ContainsFunc(result.css.nodes, func(existing cssResource) bool {
					return link.equal(existing)
				}) {
					continue
				}
				result.css.nodes = append(result.css.nodes, link)
				if link.CSSInlineRelationCalculator != nil || link.CSSLinkRelationCalculator != nil || link.DisableImplicitOrdering {
					continue
				}
				thisNode := len(result.css.nodes) - 1
				if lastLink >= 0 {
					if result.css.edgesFrom[thisNode] == nil {
						result.css.edgesFrom[thisNode] = map[int]struct{}{}
					}
					if result.css.edgesTo[lastLink] == nil {
						result.css.edgesTo[lastLink] = map[int]struct{}{}
					}
					result.css.edgesFrom[thisNode][lastLink] = struct{}{}
					result.css.edgesTo[lastLink][thisNode] = struct{}{}
				}
				lastLink = thisNode
			}
		}
		if cssEmbedder, ok := component.(CSSEmbedder); ok {
			blocks := cssEmbedder.EmbedCSS(ctx)
			lastBlock := -1
			for _, block := range blocks {
				if slices.ContainsFunc(result.css.nodes, func(existing cssResource) bool {
					return block.equal(existing)
				}) {
					continue
				}
				result.css.nodes = append(result.css.nodes, block)
				if block.CSSInlineRelationCalculator != nil || block.CSSLinkRelationCalculator != nil || block.DisableImplicitOrdering {
					continue
				}
				thisNode := len(result.css.nodes) - 1
				if lastBlock >= 0 {
					if result.css.edgesFrom[thisNode] == nil {
						result.css.edgesFrom[thisNode] = map[int]struct{}{}
					}
					if result.css.edgesTo[lastBlock] == nil {
						result.css.edgesTo[lastBlock] = map[int]struct{}{}
					}
					result.css.edgesFrom[thisNode][lastBlock] = struct{}{}
					result.css.edgesTo[lastBlock][thisNode] = struct{}{}
				}
				lastBlock = thisNode
			}
		}
		if jsLinker, ok := component.(JSLinker); ok {
			links := jsLinker.LinkJS(ctx)
			lastHeadLink, lastFootLink := -1, -1
			for _, link := range links {
				if link.PlaceInFooter {
					if slices.ContainsFunc(result.footJS.nodes, func(existing jsResource) bool {
						return link.equal(existing)
					}) {
						continue
					}
					result.footJS.nodes = append(result.footJS.nodes, link)
					if link.JSInlineRelationCalculator != nil || link.JSLinkRelationCalculator != nil || link.DisableImplicitOrdering {
						continue
					}
					thisNode := len(result.footJS.nodes) - 1
					if lastFootLink >= 0 {
						if result.footJS.edgesFrom[thisNode] == nil {
							result.footJS.edgesFrom[thisNode] = map[int]struct{}{}
						}
						if result.footJS.edgesTo[lastFootLink] == nil {
							result.footJS.edgesTo[lastFootLink] = map[int]struct{}{}
						}
						result.footJS.edgesFrom[thisNode][lastFootLink] = struct{}{}
						result.footJS.edgesTo[lastFootLink][thisNode] = struct{}{}
					}
					lastFootLink = thisNode
				} else {
					if slices.ContainsFunc(result.headJS.nodes, func(existing jsResource) bool {
						return link.equal(existing)
					}) {
						continue
					}
					result.headJS.nodes = append(result.headJS.nodes, link)
					if link.JSInlineRelationCalculator != nil || link.JSLinkRelationCalculator != nil || link.DisableImplicitOrdering {
						continue
					}
					thisNode := len(result.headJS.nodes) - 1
					if lastHeadLink >= 0 {
						if result.headJS.edgesFrom[thisNode] == nil {
							result.headJS.edgesFrom[thisNode] = map[int]struct{}{}
						}
						if result.headJS.edgesTo[lastHeadLink] == nil {
							result.headJS.edgesTo[lastHeadLink] = map[int]struct{}{}
						}
						result.headJS.edgesFrom[thisNode][lastHeadLink] = struct{}{}
						result.headJS.edgesTo[lastHeadLink][thisNode] = struct{}{}
					}
					lastHeadLink = thisNode
				}
			}
		}
		if jsEmbedder, ok := component.(JSEmbedder); ok {
			blocks := jsEmbedder.EmbedJS(ctx)
			lastHeadBlock, lastFootBlock := -1, -1
			for _, block := range blocks {
				if block.PlaceInFooter {
					if slices.ContainsFunc(result.footJS.nodes, func(existing jsResource) bool {
						return block.equal(existing)
					}) {
						continue
					}
					result.footJS.nodes = append(result.footJS.nodes, block)
					if block.JSInlineRelationCalculator != nil || block.JSLinkRelationCalculator != nil || block.DisableImplicitOrdering {
						continue
					}
					thisNode := len(result.footJS.nodes) - 1
					if lastFootBlock >= 0 {
						if result.footJS.edgesFrom[thisNode] == nil {
							result.footJS.edgesFrom[thisNode] = map[int]struct{}{}
						}
						if result.footJS.edgesTo[lastFootBlock] == nil {
							result.footJS.edgesTo[lastFootBlock] = map[int]struct{}{}
						}
						result.footJS.edgesFrom[thisNode][lastFootBlock] = struct{}{}
						result.footJS.edgesTo[lastFootBlock][thisNode] = struct{}{}
					}
					lastFootBlock = thisNode
				} else {
					if slices.ContainsFunc(result.headJS.nodes, func(existing jsResource) bool {
						return block.equal(existing)
					}) {
						continue
					}
					result.headJS.nodes = append(result.headJS.nodes, block)
					if block.JSInlineRelationCalculator != nil || block.JSLinkRelationCalculator != nil || block.DisableImplicitOrdering {
						continue
					}
					thisNode := len(result.headJS.nodes) - 1
					if lastHeadBlock >= 0 {
						if result.headJS.edgesFrom[thisNode] == nil {
							result.headJS.edgesFrom[thisNode] = map[int]struct{}{}
						}
						if result.headJS.edgesTo[lastHeadBlock] == nil {
							result.headJS.edgesTo[lastHeadBlock] = map[int]struct{}{}
						}
						result.headJS.edgesFrom[thisNode][lastHeadBlock] = struct{}{}
						result.headJS.edgesTo[lastHeadBlock][thisNode] = struct{}{}
					}
					lastHeadBlock = thisNode
				}
			}
		}
	}
	for pos, resource := range result.css.nodes {
		var linkComparer func(context.Context, CSSLink) ResourceRelationship
		var inlineComparer func(context.Context, CSSInline) ResourceRelationship
		switch res := resource.(type) {
		case CSSInline:
			linkComparer = res.CSSLinkRelationCalculator
			inlineComparer = res.CSSInlineRelationCalculator
		case CSSLink:
			linkComparer = res.CSSLinkRelationCalculator
			inlineComparer = res.CSSInlineRelationCalculator
		}
		if linkComparer == nil && inlineComparer == nil {
			continue
		}
		for compPos, comparison := range result.css.nodes {
			rel := ResourceRelationshipNeutral
			switch comp := comparison.(type) {
			case CSSInline:
				if inlineComparer == nil {
					continue
				}
				rel = inlineComparer(ctx, comp)
			case CSSLink:
				if linkComparer == nil {
					continue
				}
				rel = linkComparer(ctx, comp)
			}
			switch rel {
			case ResourceRelationshipAfter:
				if result.css.edgesFrom[pos] == nil {
					result.css.edgesFrom[pos] = map[int]struct{}{}
				}
				if result.css.edgesTo[compPos] == nil {
					result.css.edgesTo[compPos] = map[int]struct{}{}
				}
				result.css.edgesFrom[pos][compPos] = struct{}{}
				result.css.edgesTo[compPos][pos] = struct{}{}
			case ResourceRelationshipBefore:
				if result.css.edgesFrom[compPos] == nil {
					result.css.edgesFrom[compPos] = map[int]struct{}{}
				}
				if result.css.edgesTo[pos] == nil {
					result.css.edgesTo[pos] = map[int]struct{}{}
				}
				result.css.edgesFrom[compPos][pos] = struct{}{}
				result.css.edgesTo[pos][compPos] = struct{}{}
			case ResourceRelationshipNeutral:
				// do nothing, this doesn't imply dependency
			}
		}
	}
	for pos, resource := range result.headJS.nodes {
		var linkComparer func(context.Context, JSLink) ResourceRelationship
		var inlineComparer func(context.Context, JSInline) ResourceRelationship
		switch res := resource.(type) {
		case JSInline:
			linkComparer = res.JSLinkRelationCalculator
			inlineComparer = res.JSInlineRelationCalculator
		case JSLink:
			linkComparer = res.JSLinkRelationCalculator
			inlineComparer = res.JSInlineRelationCalculator
		}
		if linkComparer == nil && inlineComparer == nil {
			continue
		}
		for compPos, comparison := range result.headJS.nodes {
			rel := ResourceRelationshipNeutral
			switch comp := comparison.(type) {
			case JSInline:
				rel = inlineComparer(ctx, comp)
			case JSLink:
				rel = linkComparer(ctx, comp)
			}
			switch rel {
			case ResourceRelationshipAfter:
				if result.headJS.edgesFrom[pos] == nil {
					result.headJS.edgesFrom[pos] = map[int]struct{}{}
				}
				if result.headJS.edgesTo[compPos] == nil {
					result.headJS.edgesTo[compPos] = map[int]struct{}{}
				}
				result.headJS.edgesFrom[pos][compPos] = struct{}{}
				result.headJS.edgesTo[compPos][pos] = struct{}{}
			case ResourceRelationshipBefore:
				if result.headJS.edgesFrom[compPos] == nil {
					result.headJS.edgesFrom[compPos] = map[int]struct{}{}
				}
				if result.headJS.edgesTo[pos] == nil {
					result.headJS.edgesTo[pos] = map[int]struct{}{}
				}
				result.headJS.edgesFrom[compPos][pos] = struct{}{}
				result.headJS.edgesTo[pos][compPos] = struct{}{}
			case ResourceRelationshipNeutral:
				// do nothing, this doesn't imply dependency
			}
		}
	}
	for pos, resource := range result.footJS.nodes {
		var linkComparer func(context.Context, JSLink) ResourceRelationship
		var inlineComparer func(context.Context, JSInline) ResourceRelationship
		switch res := resource.(type) {
		case JSInline:
			linkComparer = res.JSLinkRelationCalculator
			inlineComparer = res.JSInlineRelationCalculator
		case JSLink:
			linkComparer = res.JSLinkRelationCalculator
			inlineComparer = res.JSInlineRelationCalculator
		}
		if linkComparer == nil && inlineComparer == nil {
			continue
		}
		for compPos, comparison := range result.footJS.nodes {
			rel := ResourceRelationshipNeutral
			switch comp := comparison.(type) {
			case JSInline:
				rel = inlineComparer(ctx, comp)
			case JSLink:
				rel = linkComparer(ctx, comp)
			}
			switch rel {
			case ResourceRelationshipAfter:
				if result.footJS.edgesFrom[pos] == nil {
					result.footJS.edgesFrom[pos] = map[int]struct{}{}
				}
				if result.footJS.edgesTo[compPos] == nil {
					result.footJS.edgesTo[compPos] = map[int]struct{}{}
				}
				result.footJS.edgesFrom[pos][compPos] = struct{}{}
				result.footJS.edgesTo[compPos][pos] = struct{}{}
			case ResourceRelationshipBefore:
				if result.footJS.edgesFrom[compPos] == nil {
					result.footJS.edgesFrom[compPos] = map[int]struct{}{}
				}
				if result.footJS.edgesTo[pos] == nil {
					result.footJS.edgesTo[pos] = map[int]struct{}{}
				}
				result.footJS.edgesFrom[compPos][pos] = struct{}{}
				result.footJS.edgesTo[pos][compPos] = struct{}{}
			case ResourceRelationshipNeutral:
				// do nothing, this doesn't imply dependency
			}
		}
	}
	return result
}

func sortNodesByPos[Node any](nodes []Node, a, b int) int {
	return sortNodes(nodes[a], nodes[b])
}

func sortNodes[Node any](first, second Node) int {
	switch firstNode := any(first).(type) {
	case CSSInline:
		switch secondNode := any(second).(type) {
		case CSSInline:
			firstKey := firstNode.getKey()
			secondKey := secondNode.getKey()
			if firstKey < secondKey {
				return -1
			}
			if secondKey < firstKey {
				return 1
			}
			return 0
		case CSSLink:
			return 1
		default:
			panic(fmt.Sprintf("unexpected type %T when sorting CSS resources", second))
		}
	case CSSLink:
		switch secondNode := any(second).(type) {
		case CSSInline:
			return -1
		case CSSLink:
			if firstNode.Href < secondNode.Href {
				return -1
			}
			if secondNode.Href < firstNode.Href {
				return 1
			}
			return 0
		default:
			panic(fmt.Sprintf("unexpected type %T when sorting CSS resources", second))
		}
	case JSInline:
		switch secondNode := any(second).(type) {
		case JSInline:
			firstKey := firstNode.getKey()
			secondKey := secondNode.getKey()
			if firstKey < secondKey {
				return -1
			}
			if secondKey < firstKey {
				return 1
			}
			return 0
		case JSLink:
			return 1
		default:
			panic(fmt.Sprintf("unexpected type %T when sorting JavaScript resources", second))
		}
	case JSLink:
		switch secondNode := any(second).(type) {
		case JSInline:
			return -1
		case JSLink:
			if firstNode.Src < secondNode.Src {
				return -1
			}
			if secondNode.Src < firstNode.Src {
				return 1
			}
			return 0
		default:
			panic(fmt.Sprintf("unexpected type %T when sorting JavaScript resources", second))
		}
	default:
		panic(fmt.Sprintf("unexpected type %T when sorting resources", first))
	}
}

func walkGraph[Node any](_ context.Context, resources graph[Node]) ([]Node, error) {
	noParents := make([]int, 0, len(resources.nodes))
	results := make([]Node, 0, len(resources.nodes))
	for pos := range resources.nodes {
		edges, ok := resources.edgesFrom[pos]
		if !ok {
			noParents = append(noParents, pos)
			continue
		}
		if len(edges) < 1 {
			noParents = append(noParents, pos)
			continue
		}
	}
	slices.SortFunc(noParents, func(a, b int) int {
		return sortNodesByPos(resources.nodes, a, b)
	})
	for len(noParents) > 0 {
		pos := noParents[0]
		node := resources.nodes[pos]
		noParents = noParents[1:]
		results = append(results, node)
		var noParentsChanged bool
		for child := range resources.edgesTo[pos] {
			delete(resources.edgesFrom[child], pos)
			delete(resources.edgesTo[pos], child)
			if len(resources.edgesFrom[child]) < 1 {
				delete(resources.edgesFrom, child)
				noParents = append(noParents, child)
				noParentsChanged = true
			}
			if len(resources.edgesTo[pos]) < 1 {
				delete(resources.edgesTo, pos)
			}
		}
		if noParentsChanged {
			slices.SortFunc(noParents, func(a, b int) int {
				return sortNodesByPos(resources.nodes, a, b)
			})
		}
	}
	if len(resources.edgesTo) > 0 || len(resources.edgesFrom) > 0 {
		var edgesTo, edgesFrom, resourceIDs []string
		for k, v := range resources.edgesTo {
			var vals []string
			for val := range v {
				vals = append(vals, strconv.Itoa(val))
			}
			edgesTo = append(edgesTo, fmt.Sprintf("%d:%s", k, strings.Join(vals, ",")))
		}
		for k, v := range resources.edgesFrom {
			var vals []string
			for val := range v {
				vals = append(vals, strconv.Itoa(val))
			}
			edgesFrom = append(edgesFrom, fmt.Sprintf("%d:%s", k, strings.Join(vals, ",")))
		}
		for _, v := range resources.nodes {
			switch res := any(v).(type) {
			case CSSLink:
				resourceIDs = append(resourceIDs, fmt.Sprintf("CSSLink(%s)", res.Href))
			case CSSInline:
				resourceIDs = append(resourceIDs, fmt.Sprintf("CSSInline(%s)", res.TemplatePath))
			case JSLink:
				resourceIDs = append(resourceIDs, fmt.Sprintf("JSLink(%s)", res.Src))
			case JSInline:
				resourceIDs = append(resourceIDs, fmt.Sprintf("JSInline(%s)", res.TemplatePath))
			default:
				resourceIDs = append(resourceIDs, fmt.Sprintf("UnidentifiedResource(%T)", res))
			}
		}
		return results, fmt.Errorf("%w: edges_to=[%s], edges_from=[%s], resources=[%s]", ErrResourceCycle, strings.Join(edgesTo, "; "), strings.Join(edgesFrom, "; "), strings.Join(resourceIDs, ", "))
	}
	return results, nil
}
