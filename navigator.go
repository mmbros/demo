package main

import "github.com/mmbros/demo/treepath"

// Navigator implements treepath.Element interface for Node
type Navigator struct{ node *Node }

// Parent returns the parent element.
// It returns nil in case of root element.
func (nav Navigator) Parent() treepath.Element {
	par := Navigator{nav.node.parent}
	return treepath.Element(&par)
}

func (nav Navigator) Children() []treepath.Element {
	elements := make([]treepath.Element, len(nav.node.children))
	for j, c := range nav.node.children {
		child := Navigator{c}
		elements[j] = &child
	}
	return elements
}

func (nav Navigator) String() string { return nav.node.name }

// MatchTag returns true if the ...
func (nav Navigator) MatchTag(tag string) bool {
	return nav.node.name == tag
}
