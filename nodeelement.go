package main

import "github.com/mmbros/demo/treepath"

// NodeElement implements treepath.Element interface for Node
type NodeElement struct{ *Node }

// Parent returns the parent element.
// It returns nil in case of root element.
func (e NodeElement) Parent() treepath.Element {
	par := NodeElement{e.parent}
	return treepath.Element(&par)
}

// Childre returns the children element of the current node
func (e NodeElement) Children() []treepath.Element {
	elements := make([]treepath.Element, len(e.children))
	for j, c := range e.children {
		child := NodeElement{c}
		elements[j] = &child
	}
	return elements
}

func (e NodeElement) String() string { return e.name }

// MatchTag returns true if ...
func (e NodeElement) MatchTag(tag string) bool {
	return e.name == tag
}

// MatchTagText returns true if ...
func (e NodeElement) MatchTagText(tag, text string) bool {
	return false
}

// MatchAttr returns true if ...
func (e NodeElement) MatchAttr(attr string) bool {
	return false
}

// MatchAttrText returns true if ...
func (e NodeElement) MatchAttrText(attr, text string) bool {
	return false
}
