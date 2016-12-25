package main

import (
	"fmt"

	"github.com/mmbros/demo/treepath"
)

type Node struct {
	name     string
	parent   *Node
	children []*Node
}

/*
func (n *Node) Parent() treepath.Element {
	return treepath.Element(n.parent)
}

func (n *Node) ElementChildren() []treepath.Element {
	elements := make([]treepath.Element, len(n.children))
	for j, c := range n.children {
		elements[j] = c
	}
	return elements
}
*/
func (n *Node) Add(child *Node) {
	child.parent = n
	n.children = append(n.children, child)
}

func NewElement(name string) *Node {
	return &Node{name: name}
}
func (n *Node) PrintTree(prefix string) {
	fmt.Println(prefix + n.name)
	prefix = prefix + "-  "
	for _, c := range n.children {
		c.PrintTree(prefix)
	}

}

//-----------------------------------------------------------------------------

func main() {
	root := NewElement("<ROOT>")
	A := NewElement("DIV")
	B := NewElement("SPAN")
	root.Add(A)
	root.Add(B)
	A.Add(NewElement("P"))
	A2 := NewElement("P")
	A2.Add(NewElement("SPAN"))
	A.Add(A2)
	A.Add(NewElement("P"))

	root.PrintTree("")

	//path, err := CompilePath("SPAN") // ok
	//path, err := CompilePath("DIV/P") // ok
	path, err := treepath.CompilePath("DIV//SPAN")
	if err != nil {
		fmt.Println(err)
	}
	nav := NodeElement{root}

	fmt.Printf("root: %p\n", root)
	fmt.Printf("nav : %p\n", nav)

	res := path.FindElements(&nav)
	for j, e := range res {
		fmt.Printf("%d) %v\n", j, e)
	}

}
