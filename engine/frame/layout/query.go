package layout

import "github.com/npillmayer/tyse/engine/frame"

type Q struct {
	root       *frame.PrincipalBox
	predicates []QueryPredicate
}

type QueryPredicate func(c frame.Container) frame.Container

func Query(root *frame.PrincipalBox, pred QueryPredicate) *Q {
	return &Q{
		root:       root,
		predicates: make([]QueryPredicate, 1),
	}
}

func (q *Q) AllBoxes() []frame.Box {
	return nil
}

func (q *Q) All() []frame.Container {
	return nil
}
