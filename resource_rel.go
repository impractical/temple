package temple

// ResourceRelationship controls the relationship between two resources. It's
// used to control the order in which CSS and JavaScript resources are rendered
// to the page.
type ResourceRelationship string

const (
	// ResourceRelationshipAfter indicates that the resource should be
	// rendered after the resource it's being compared to.
	ResourceRelationshipAfter ResourceRelationship = "after"

	// ResourceRelationshipBefore indicates that the resource should be
	// rendered before the resource it's being compared to.
	ResourceRelationshipBefore ResourceRelationship = "before"

	// ResourceRelationshipNeutral indicates that the resource has no
	// restrictions about where it's rendered in relation to the resource
	// it's being compared to. Generally, for performance reasons, it's
	// preferred to omit the relationship calculation function entirely
	// rather than returning a ResourceRelationshipNeutral value, but if
	// the function could return other values when compared to other
	// resources, ResourceRelationshipNeutral is necessary to be able to
	// indicate that no relationship exists between these particular
	// resources.
	ResourceRelationshipNeutral ResourceRelationship = "neutral"
)
