package layout

// This module should have knowledge about:
// - which mini-hierarchy of boxes to create for each HTML element
// - which context the element should span for its children

import (
	"errors"

	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/tree"
	"golang.org/x/net/html"
)

var errDOMRootIsNull = errors.New("DOM root is null")
var errDOMNodeNotSuitable = errors.New("DOM node is not suited for layout")

// BuildBoxTree creates a render box tree from a styled tree.
func BuildBoxTree(domRoot *dom.W3CNode) (Container, error) {
	if domRoot == nil {
		return nil, errDOMRootIsNull
	}
	domWalker := domRoot.Walk()
	T().Debugf("Creating box tree")
	dom2box := newAssoc()
	createBoxForEach := prepareBoxCreator(dom2box)
	future := domWalker.TopDown(createBoxForEach).Promise() // start asynchronous traversal
	renderNodes, err := future()                            // wait for top-down traversal to finish
	if err != nil {
		return nil, err
	}
	T().Infof("Walker returned %d render nodes", len(renderNodes))
	/*
		for _, rnode := range renderNodes {
			n := TreeNodeAsPrincipalBox(rnode)
			T().Infof("  node for %s", n.domNode.NodeName())
		}
	*/
	T().Infof("dom2box dict contains %d entries", dom2box.Length())
	//T().Errorf("domRoot/2 = %s", dbgNodeString(domRoot))
	boxRoot, ok := dom2box.Get(domRoot)
	//T().Errorf("box for domRoot = %v", boxRoot)
	if !ok {
		T().Errorf("No box created for root style node")
	}
	return boxRoot, nil
}

// prepareBocCreator is an action function for concurrent tree-traversal.
func prepareBoxCreator(dict *domToBoxAssoc) tree.Action {
	dom2box := dict
	action := func(node *tree.Node, parentNode *tree.Node, chpos int) (*tree.Node, error) {
		domnode, err := dom.NodeFromTreeNode(node)
		if err != nil {
			T().Errorf("action 1: %s", err.Error())
			return nil, err
		}
		var parent *dom.W3CNode
		if parentNode != nil {
			parent, err = dom.NodeFromTreeNode(parentNode)
			if err != nil {
				T().Errorf("action 2: %s", err.Error())
				return nil, err
			}
		}
		return makeBoxNode(domnode, parent, chpos, dom2box)
	}
	return action
}

func makeBoxNode(domnode *dom.W3CNode, parent *dom.W3CNode, chpos int, dom2box *domToBoxAssoc) (
	*tree.Node, error) {
	//
	T().Infof("making box for %s", domnode.NodeName())
	box := NewBoxForDOMNode(domnode)
	if box == nil { // legit, e.g. for "display:none"
		return nil, nil // will not descend to children of domnode
	}
	T().Infof("remembering %d/%s", domnode.NodeType(), domnode.NodeName())
	dom2box.Put(domnode, box) // associate the styled tree node to this box
	if !domnode.IsDocument() {
		if parentNode := domnode.ParentNode(); parentNode != nil {
			parent := parentNode.(*dom.W3CNode)
			parentbox, found := dom2box.Get(parent)
			if found {
				T().Debugf("adding new box %s node to parent %s\n", box, parentbox)
				p := parentbox.(*PrincipalBox)
				var err error
				switch b := box.(type) {
				case *PrincipalBox:
					b.ChildInx = uint32(chpos)
					err = p.AddChild(b)
				case *TextBox:
					b.ChildInx = uint32(chpos)
					err = p.AddTextChild(b)
				default:
					T().Errorf("Unknown box type for %v", box)
				}
				if err != nil {
					T().Errorf(err.Error())
				}
				_, ok := p.Child(0)
				if !ok {
					T().Errorf("Parent has no child!")
				}
			}
		}
	}
	//possiblyCreateMiniHierarchy(box)
	return box.TreeNode(), nil
}

// ----------------------------------------------------------------------

// NewBoxForDOMNode creates an adequately initialized box for a given DOM node.
func NewBoxForDOMNode(domnode *dom.W3CNode) Container {
	if domnode.NodeType() == html.TextNode {
		tbox := newTextBox(domnode)
		// TODO find index within parent
		// and set #ChildInx
		return tbox
	}
	// document or element node
	outerMode, innerMode := DisplayModesForDOMNode(domnode)
	if outerMode == NoMode || outerMode == DisplayNone {
		return nil // do not produce box for illegal mode or for display = "none"
	}
	pbox := newPrincipalBox(domnode, outerMode, innerMode)
	pbox.prepareAnonymousBoxes()
	// TODO find index within parent
	// and set #ChildInx
	return pbox
}

func possiblyCreateMiniHierarchy(pbox *PrincipalBox) {
	htmlnode := pbox.domNode.HTMLNode()
	//propertyMap := styler.ComputedStyles()
	switch htmlnode.Data {
	case "li":
		//markertype, _ := style.GetCascadedProperty(c.DOMNode, "list-style-type", toStyler)
		markertype := pbox.domNode.ComputedStyles().GetPropertyValue("list-style-type")
		if markertype != "none" {
			//markerbox := newContainer(BlockMode, FlowMode)
			// TODO: fill box with correct marker symbol
			//pbox.Add(markerbox)
			T().Debugf("need marker for principal box")
		}
	}
}
