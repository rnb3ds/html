package html

// ============================================================================
// ContentNode Interface (for Scorer abstraction)
// ============================================================================

// NodeAttr represents a single HTML node attribute.
type NodeAttr struct {
	Key   string
	Value string
}

// ContentNode provides an abstraction over HTML node structure
// for content scoring purposes. This interface hides the internal
// golang.org/x/net/html dependency from public API consumers,
// allowing custom Scorers to be implemented without importing
// the internal HTML parser package.
//
// The interface provides read-only access to node properties
// needed for content quality assessment and filtering.
type ContentNode interface {
	// Type returns the node type as a string.
	// Possible values: "element", "text", "comment", "document", "doctype",
	// "error", "raw", or "unknown".
	Type() string

	// Data returns the element tag name for element nodes (e.g., "div", "p"),
	// or the text content for text nodes.
	Data() string

	// AttrValue returns the value of the attribute with the given key,
	// or an empty string if the attribute does not exist.
	AttrValue(key string) string

	// Attrs returns all attributes of the node.
	Attrs() []NodeAttr

	// FirstChild returns the first child node, or nil if none.
	FirstChild() ContentNode

	// NextSibling returns the next sibling node, or nil if none.
	NextSibling() ContentNode

	// Parent returns the parent node, or nil if this is the root.
	Parent() ContentNode
}
