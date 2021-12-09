package boxtree

import (
	"github.com/npillmayer/tyse/core/option"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/tree"
)

// reorderBoxTree reorders box nodes of a box-tree to account for
// "position" and "float" CSS properties.
//
// Currently this function moves boxes with positions 'fixed' or 'absolute' out of
// the normal DOM hierarchy and re-attaches them to the document root or an ancestor
// with non-static positioning, respectively.
//
// In a future version, CSS regions should be supported as well.
//
func reorderBoxTree(boxRoot *PrincipalBox) error {
	tracer().Infof("=========== box REORDER ========================================")
	if boxRoot == nil {
		return nil
	}
	walker := tree.NewWalker(boxRoot.TreeNode())
	_, err := walker.DescendentsWith(reposition).AncestorWith(anchor).Promise()()
	return err
}

// Tree filter predicate: box has position "fixed", "absolute" or is a float.
func reposition(node *tree.Node, unused *tree.Node) (match *tree.Node, err error) {
	pbox := TreeNodeAsPrincipalBox(node)
	if pbox != nil {
		pos := css.PositionOption(pbox.domNode)
		if nonflow, _ := pos.Match(option.Of{
			option.None:            false,
			css.PositionFixed:      true,
			css.PositionAbsolute:   true,
			css.PositionFloatLeft:  true,
			css.PositionFloatRight: true,
			option.Some:            false,
		}); nonflow.(bool) {
			tracer().Debugf("box has to be re-ordered: %s (%v)", boxname(pbox), pos)
			match = pbox.TreeNode()
		}
	}
	return
}

// Tree filter predicate with side effect: attaches node to anchor, if suited.
func anchor(anchorCandidate *tree.Node, node *tree.Node) (match *tree.Node, err error) {
	if node == nil || anchorCandidate == nil {
		panic("one of node, anchor is nil")
	}
	positionedChild := node.Payload.(frame.Container)
	possibleAnchor := anchorCandidate.Payload.(frame.Container)
	if positionedChild.Type() != TypePrincipal || possibleAnchor.Type() != TypePrincipal {
		return
	}
	tracer().Debugf("trying to re-attach %s node", boxname(positionedChild))
	tracer().Debugf("   candidate anchor is %s", boxname(possibleAnchor))
	var anchor *PrincipalBox
	anchorCandidateIsDocRoot := func(interface{}) (interface{}, error) {
		return possibleAnchor.DOMNode().NodeName() == "#document", nil
	}
	anchorCandidateIsPositioned := func(interface{}) (interface{}, error) {
		ancpos := css.PositionOption(possibleAnchor.DOMNode())
		ok, _ := ancpos.Match(option.Of{
			option.None:          false,
			css.PositionAbsolute: true,
			css.PositionFixed:    true,
			css.PositionRelative: true,
			css.PositionSticky:   true,
			option.Some:          false,
		})
		return ok.(bool), nil
	}
	anchorCandidateIsFlowRoot := func(interface{}) (interface{}, error) {
		return possibleAnchor.Context().IsFlowRoot(), nil
	}
	position := css.PositionOption(positionedChild.DOMNode())
	found, err := position.Match(option.Of{
		css.PositionFixed:      anchorCandidateIsDocRoot,    // test for document root
		css.PositionAbsolute:   anchorCandidateIsPositioned, // test for out-of-flow position
		css.PositionFloatLeft:  anchorCandidateIsFlowRoot,   // test for flow-root
		css.PositionFloatRight: anchorCandidateIsFlowRoot,   // test for flow-root
	})
	if err != nil || !(found.(bool)) {
		return
	}
	anchor = possibleAnchor.(*PrincipalBox)
	if anchor != nil {
		positionedChild.TreeNode().Isolate()
		anchor.AppendChild(positionedChild.(*PrincipalBox)) // TODO will lose ordering !
	}
	return
}
