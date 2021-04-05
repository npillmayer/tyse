package otlyt

import "github.com/npillmayer/tyse/core/font/ot"

type Feature interface {
	Tag() ot.Tag
	Apply([]rune, int) int
}

type feature struct {
	tag ot.Tag
	nav ot.Navigator
}

func (f feature) Tag() ot.Tag {
	return f.tag
}

func New(otf *ot.Font, tag ot.Tag) Feature {
	return nil
}
