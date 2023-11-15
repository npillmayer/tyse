package css

import (
	"strings"

	"github.com/npillmayer/tyse/engine/dom/style"
)

// position is an enum type for the CSS position property.
type position uint16

// Enum values for type Position
const (
	positionUnset    position = iota
	positionStatic            // CSS static (default)
	positionRelative          // CSS relative
	positionAbsolute          // CSS absolute
	positionFixed             // CSS fixed
	//PositionFloatLeft           // CSS float property
	//PositionFloatRight          // CSS float property
	//PositionSticky              // CSS sticky, currently mapped to relative
)

// PositionT is an option type for CSS positions.
type PositionT struct {
	offsets []PositionOffset
	kind    position
}

type PositionOffset struct {
	Dim DimenT
	Dir PosDir
}

// PosDir is either Top, Right, Bottom or Left.
type PosDir uint8

const (
	Top PosDir = iota
	Right
	Bottom
	Left
)

// NormalizeOffsets normalizes offset properties (Top, Right, Bottom, Left) into
// a 4-way slice, ordered by PDir. Invalid PDir-s are silently dropped.
func NormalizeOffsets(offsets []PositionOffset) []PositionOffset {
	norm := make([]PositionOffset, 4)
	for i := Top; i <= Left; i++ {
		norm[i].Dir = i
	}
	for _, o := range offsets {
		if o.Dir >= Top && o.Dir <= Left {
			norm[int(o.Dir)] = o
		}
	}
	return norm
}

// ZeroOffsets returns (Top, Right, Bottom, Left) = (0, 0, 0, 0)
func ZeroOffsets() []PositionOffset {
	zeros := make([]PositionOffset, 4)
	for i := Top; i <= Left; i++ {
		zeros[i].Dir = i
	}
	return zeros
}

/*
type PositionT
	= Undefined
	| Static
	| Relative top right bottom left
	| Absolute top right bottom left
	| Fixed top right bottom left
*/

// Static creates a CSS position of value `static`.
func Static() PositionT {
	return PositionT{kind: positionStatic}
}

// Relative creates a CSS position of value `relative`, given optional offsets.
// offsets may be provied partially or none at all.
func Relative(offsets []PositionOffset) PositionT {
	return PositionT{kind: positionRelative, offsets: NormalizeOffsets(offsets)}
}

// Absolute creates a CSS position of value `absolute`, given optional offsets.
// offsets may be provied partially or none at all.
func Absolute(offsets []PositionOffset) PositionT {
	return PositionT{kind: positionAbsolute, offsets: NormalizeOffsets(offsets)}
}

// Fixed creates a CSS position of value `fixed`, given optional offsets.
// offsets may be provied partially or none at all.
func Fixed(offsets []PositionOffset) PositionT {
	return PositionT{kind: positionFixed, offsets: NormalizeOffsets(offsets)}
}

var positionMap map[position]string = map[position]string{
	positionStatic:   "static",
	positionRelative: "relative",
	positionAbsolute: "absolute",
	positionFixed:    "fixed",
	//PositionFloatLeft:  "float",
	//PositionFloatRight: "float",
	//PositionSticky:     "sticky",
}

var positionStringMap map[string]position = map[string]position{
	"static":   positionStatic,
	"relative": positionRelative,
	"absolute": positionAbsolute,
	"fixed":    positionFixed,
	//"float":    PositionFloatLeft,
	//"sticky":   PositionSticky,
}

// Position returns an optional position type from a property string.
// It will never return an error, even with illegal input, but instead will then
// return an unset position.
func Position(p style.Property) PositionT {
	p = style.Property(strings.ToLower(string(p)))
	switch p {
	case style.NullStyle:
		return PositionT{}
	case "static":
		return Static()
	case "relative":
		return Relative(nil)
	case "absolute":
		return Absolute(nil)
	case "fixed":
		return Fixed(nil)
	}
	return PositionT{}
}

// ---------------------------------------------------------------------------

func (p PositionT) Match() *PMatcher {
	return &PMatcher{pos: p}
}

type PMatcher struct {
	pos PositionT
}

func (m *PMatcher) IsKind(p PositionT) *PMatcher {
	switch {
	case p.kind == positionUnset && m.pos.kind == positionUnset:
		return m
	case p.kind == positionStatic && m.pos.kind == positionStatic:
		return m
	case p.kind == positionRelative && m.pos.kind == positionRelative:
		return m
	case p.kind == positionAbsolute && m.pos.kind == positionAbsolute:
		return m
	case p.kind == positionFixed && m.pos.kind == positionFixed:
		return m
	}
	return nil
}

func (m *PMatcher) Absolute(o *[]PositionOffset) *PMatcher {
	if m.pos.kind == positionAbsolute {
		if o != nil {
			*o = m.pos.offsets
		}
		return m
	}
	return nil
}

func (m *PMatcher) Relative(o *[]PositionOffset) *PMatcher {
	if m.pos.kind == positionRelative {
		if o != nil {
			*o = m.pos.offsets
		}
		return m
	}
	return nil
}

func (m *PMatcher) Fixed(o *[]PositionOffset) *PMatcher {
	if m.pos.kind == positionFixed {
		if o != nil {
			*o = m.pos.offsets
		}
		return m
	}
	return nil
}

// --- Expression matching ---------------------------------------------------

type PositionPatterns[T any] struct {
	Unset    T
	Static   T
	Absolute T
	Relative T
	Fixed    T
	Default  T
}

func PositionPattern[T any](p PositionT) *PMatchExpr[T] {
	return &PMatchExpr[T]{pos: p}
}

// PMatchExpr is part of pattern matching for PositionT types and intended to be instantiated
// using `PosPattern()` only.
type PMatchExpr[T any] struct {
	pos PositionT
}

func (m *PMatchExpr[T]) OneOf(patterns PositionPatterns[T]) T {
	switch {
	case m.pos.kind == positionUnset:
		return patterns.Unset
	case m.pos.kind == positionAbsolute:
		return patterns.Absolute
	case m.pos.kind == positionRelative:
		return patterns.Relative
	case m.pos.kind == positionFixed:
		return patterns.Fixed
	}
	return patterns.Default
}

func (m *PMatchExpr[T]) With(o *[]PositionOffset) *PMatchExpr[T] {
	if o != nil {
		*o = m.pos.offsets
	}
	return m
}

func (m *PMatchExpr[T]) Const(x T) T {
	return x
}

// ---------------------------------------------------------------------------

// IsUnset returns true if p is unset.
func (p PositionT) IsUnset() bool {
	return p.kind == positionUnset
}

// IsRelative returns true if p represents a valid relative position.
func (p PositionT) IsRelative() bool {
	return p.kind == positionRelative
}

// IsAbsolute returns true if d represents a valid absolute position.
func (p PositionT) IsAbsolute() bool {
	return p.kind == positionUnset
}

// IsFixed returns true if d represents a fixed position.
func (p PositionT) IsFixed() bool {
	return p.kind == positionFixed
}
