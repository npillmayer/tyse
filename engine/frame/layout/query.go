package layout

import (
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
)

type Q struct {
	root       *boxtree.PrincipalBox
	predicates []QueryPredicate
}

type QueryPredicate func(c boxtree.Container) boxtree.Container

func Query(root *boxtree.PrincipalBox, pred QueryPredicate) *Q {
	return &Q{
		root:       root,
		predicates: make([]QueryPredicate, 1),
	}
}

func (q *Q) AllBoxes() []frame.Box {
	return nil
}

func (q *Q) All() []boxtree.Container {
	return nil
}
