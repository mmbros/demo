package treepath

import (
	"encoding/xml"
	"fmt"
	"testing"
)

// ----------------------------------------------------------------------------

type Node struct {
	Name     string  `xml:"name,attr"`
	Class    string  `xml:"class,attr"`
	Parent   *Node   `xml:"-"`
	Children []*Node `xml:"node"`
}

const xmlNodes = `
<doc>
	<node name="html">
		<node name="head">
			<node name="title" />
		</node>
		<node name="body">
			<node name="h1" class="title" />
			<node name="div" class="content">
				<node name="p" class="summary" />
				<node name="ul">
					<node name="li" />
					<node name="li" />
				</node>
				<node name="p" />
			</node>
			<node name="div" class="footer">
				<node name="p" />
				<node name="p" />
				<node name="div">
					<node name="p" />
					<node name="p">
						<node name="span" />
					</node>
				</node>
			</node>
		</node>
	</node>
</doc>
`

type test struct {
	path   string
	result interface{}
}

type errorResult string

var tests = []test{

	// basic queries
	{"./html", "html"},
	{"./html/head", "head"},
	{"./html/head/title", "title"},
	{"./html/head/title/tag", nil},

	// descendant queries
	{"//ul", "ul"},
	{"//li", []string{"li", "li"}},
	{"//ul/li", []string{"li", "li"}},
	{".//li", []string{"li", "li"}},
	{"./html/body//li", []string{"li", "li"}},
	{"./html//div//li", []string{"li", "li"}},

	// positional queries
	{"./html//div[1]/ul", "ul"},
	{"//div[2]/ul", nil},
	{"//p[2]/span", "span"},
	{"//p[-1]/span", "span"},
	{"./html/body/div[2]/div/p[2]/span", "span"},
}

func (n *Node) printTree(prefix string) {
	fmt.Println(prefix + n.Name)
	prefix = prefix + "-  "
	for _, c := range n.Children {
		c.printTree(prefix)
	}
}

// getRoot returns the root of the nodes unmarshalled from xmlNodes.
func getRoot() (*Node, error) {
	var fnSetParent func(*Node)
	root := &Node{}

	// unmarshal data
	err := xml.Unmarshal([]byte(xmlNodes), root)
	if err != nil {
		return nil, err
	}

	// set the parent property of the nodes
	fnSetParent = func(parent *Node) {
		for _, child := range parent.Children {
			child.Parent = parent
			fnSetParent(child)
		}
	}
	fnSetParent(root)

	return root, nil
}

// ----------------------------------------------------------------------------

// NodeElement implements treepath.Element interface for Node
type NodeElement struct{ *Node }

// Parent returns the parent element.
// It returns nil in case of root element.
func (e NodeElement) Parent() Element {
	par := NodeElement{e.Node.Parent}
	return Element(&par)
}

// Childre returns the children element of the current node
func (e NodeElement) Children() []Element {
	elements := make([]Element, len(e.Node.Children))
	for j, c := range e.Node.Children {
		child := NodeElement{c}
		elements[j] = &child
	}
	return elements
}

func (e NodeElement) String() string { return e.Name }

// MatchTag returns true if ...
func (e NodeElement) MatchTag(tag string) bool {
	return e.Name == tag
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

// ----------------------------------------------------------------------------

func findNodes(path Path, root *Node) []*Node {
	elements := path.FindElements(&NodeElement{root})
	if elements == nil || len(elements) == 0 {
		return nil
	}
	nodes := make([]*Node, len(elements))
	for j, e := range elements {
		nodes[j] = e.(*NodeElement).Node
	}
	return nodes
}

// ----------------------------------------------------------------------------

func fail(t *testing.T, test test) {
	t.Errorf("treepath: failed test '%s'\n", test.path)
}

func TestPath(t *testing.T) {

	root, err := getRoot()
	if err != nil {
		t.Fatalf("getRoot error: %v", err)
	}

	for _, test := range tests {
		path, err := CompilePath(test.path)
		if err != nil {
			if r, ok := test.result.(errorResult); !ok || err.Error() != string(r) {
				fail(t, test)
			}
			continue
		}

		nodes := findNodes(path, root)
		//t.Logf("%s -> %d items", test.path, len(nodes))
		//for j, e := range nodes {
		//t.Logf("%d) %v\n", j, e)
		//}

		switch s := test.result.(type) {
		case errorResult:
			fail(t, test)
		case nil:
			if len(nodes) != 0 {
				fail(t, test)
			}
		case string:
			if nodes == nil || len(nodes) != 1 || nodes[0].Name != s {
				fail(t, test)
			}
		case []string:
			if nodes == nil || len(nodes) != len(s) {
				fail(t, test)
				continue
			}
			for i := 0; i < len(nodes); i++ {
				if nodes[i].Name != s[i] {
					fail(t, test)
					break
				}
			}
		}
	}
}
