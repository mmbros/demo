package treepath

import (
	"fmt"

	"github.com/mmbros/demo/queue"
)

// ----------------------------------------------------------------------------

// selectSelf selects the current element into the candidate list.
type selectSelf struct{}

func (s *selectSelf) apply(e Element, p *pather) {

	fmt.Println("selectSelf:", e.String())
	p.candidates = append(p.candidates, e)
}

// ----------------------------------------------------------------------------

// selectParent selects the element's parent into the candidate list.
type selectParent struct{}

func (s *selectParent) apply(e Element, p *pather) {

	if parent := e.Parent(); parent != nil {
		fmt.Println("selectParent of ", e.String(), ": ", parent.String())
		p.candidates = append(p.candidates, parent)
	}
}

// ----------------------------------------------------------------------------

// selectChildren selects the element's child elements into the
// candidate list.
type selectChildren struct{}

func (s *selectChildren) apply(e Element, p *pather) {
	for _, child := range e.Children() {

		fmt.Println("selectChildren of ", e.String(), ": ", child.String())
		p.candidates = append(p.candidates, child)
	}
}

// ----------------------------------------------------------------------------

// selectDescendants selects all descendant child elements
// of the element into the candidate list.
type selectDescendants struct{}

func (s *selectDescendants) apply(e Element, p *pather) {
	q := queue.NewFifo(0)

	for q.Push(e); q.Len() > 0; {
		e := q.Pop().(Element)

		fmt.Println("selectDescendant: ", e.String())
		p.candidates = append(p.candidates, e)
		for _, c := range e.Children() {
			q.Push(c)
		}
	}
}

// ----------------------------------------------------------------------------

// selectChildrenByTag selects into the candidate list all child
// elements of the element having the specified tag.
type selectChildrenByTag struct {
	tag string
}

func newSelectChildrenByTag(tag string) *selectChildrenByTag {
	return &selectChildrenByTag{tag}
}

func (s *selectChildrenByTag) apply(e Element, p *pather) {
	for _, c := range e.Children() {
		if c.MatchTag(s.tag) {
			fmt.Printf("selectChildrenByTag: parent %s, child %s\n", e.String(), c.String())
			p.candidates = append(p.candidates, c)
		}
	}
}

// ----------------------------------------------------------------------------

// selectByTag selects into the candidate list all child
// elements of the element having the specified tag.
type selectByTag struct {
	tag string
}

func newSelectByTag(tag string) *selectByTag {
	return &selectByTag{tag}
}

func (s *selectByTag) apply(e Element, p *pather) {
	if e.MatchTag(s.tag) {
		fmt.Println("selectByTag of ", e.String())
		p.candidates = append(p.candidates, e)
	}
}
