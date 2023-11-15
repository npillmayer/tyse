package css

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/npillmayer/tyse/core/dimen"
	. "github.com/npillmayer/tyse/core/percent"
	"github.com/npillmayer/tyse/engine/dom/style"
)

const (
	dimenUnset uint32 = 0

	dimenAbsolute uint32 = 0x0001
	dimenAuto     uint32 = 0x0002
	dimenInherit  uint32 = 0x0003
	dimenInitial  uint32 = 0x0004
	kindMask      uint32 = 0x000f

	// Flags for content dependent dimensions
	DimenContentMax uint32 = 0x0010
	DimenContentMin uint32 = 0x0020
	DimenContentFit uint32 = 0x0030
	contentMask     uint32 = 0x00f0

	dimenEM      uint32 = 0x0100
	dimenEX      uint32 = 0x0200
	dimenCH      uint32 = 0x0300
	dimenREM     uint32 = 0x0400
	dimenVW      uint32 = 0x0500
	dimenVH      uint32 = 0x0600
	dimenVMIN    uint32 = 0x0700
	dimenVMAX    uint32 = 0x0800
	dimenPercent uint32 = 0x0900
	relativeMask uint32 = 0xff00
)

// DimenT is an option type for CSS dimensions.
type DimenT struct {
	d       dimen.DU
	percent Percent
	flags   uint32
}

/*
type DimenT
	= Auto
	| Inherit
	| Initial
	| JustDimen dimen
	| Percentage Percent
	| ViewRel unit
	| FontRel unit
	| ContentRel Min N
	| ContentRel Max N
*/

func Auto() DimenT {
	return DimenT{flags: dimenAuto}
}

func Inherit() DimenT {
	return DimenT{flags: dimenInherit}
}

func Initial() DimenT {
	return DimenT{flags: dimenInitial}
}

// JustDimen creates a CSS dimension with a fixed value of x.
func JustDimen(x dimen.DU) DimenT {
	return DimenT{d: x, flags: dimenAbsolute}
}

// Percentage creates a CSS dimension with a %-relative value.
func Percentage(n Percent) DimenT {
	return DimenT{percent: n, flags: dimenPercent}
}

// DimenOption returns an optional dimension type from a property string.
// It will never return an error, even with illegal input, but instead will then
// return an unset dimension.
func DimenOption(p style.Property) DimenT {
	switch p {
	case style.NullStyle:
		return DimenT{}
	case "auto":
		return DimenT{flags: dimenAuto}
	case "initial":
		return DimenT{flags: dimenInitial}
	case "inherit":
		return DimenT{flags: dimenInherit}
	case "fit-content":
		return DimenT{flags: DimenContentFit}
	}
	d, err := ParseDimen(string(p))
	if err != nil {
		tracer().Debugf("dimension option from property '%s': %v", p, err)
		return DimenT{}
	}
	return d
}

// ---------------------------------------------------------------------------

func (d DimenT) Match() *DMatcher {
	return &DMatcher{dimen: d}
}

type DMatcher struct {
	dimen DimenT
}

func (m *DMatcher) IsKind(d DimenT) *DMatcher {
	switch {
	case (m.dimen.flags & kindMask) == (d.flags & kindMask):
		return m
	case (m.dimen.flags&relativeMask > 0) && (d.flags&relativeMask > 0):
		if (m.dimen.flags&dimenPercent > 0) != (d.flags&dimenPercent > 0) {
			return nil
		}
		return m
	case (m.dimen.flags&contentMask > 0) && (d.flags&contentMask > 0):
		return m
	}
	return nil
}

func (m *DMatcher) Unset() *DMatcher {
	if m == nil || m.dimen.flags == dimenUnset {
		return m
	}
	return nil
}

func (m *DMatcher) Just(du *dimen.DU) *DMatcher {
	if m.dimen.flags&dimenAbsolute > 0 {
		if du != nil {
			*du = m.dimen.d
		}
		return m
	}
	return nil
}

func (m *DMatcher) Percentage(p *Percent) *DMatcher {
	if m.dimen.flags&dimenPercent > 0 {
		if p != nil {
			*p = m.dimen.percent
		}
		return m
	}
	return nil
}

// --- Expression matching ---------------------------------------------------

//type DimenPatterns[T any] map[*MatchExpr[T]]T
type DimenPatterns[T any] struct {
	Unset   T
	Auto    T
	Inherit T
	Initial T
	Just    T
	Default T
}

func DimenPattern[T any](d DimenT) *DMatchExpr[T] {
	return &DMatchExpr[T]{dimen: d}
}

type DMatchExpr[T any] struct {
	dimen DimenT
}

func (m *DMatchExpr[T]) OneOf(patterns DimenPatterns[T]) T {
	switch {
	case m.dimen.flags == dimenUnset:
		return patterns.Unset
	case m.dimen.flags&dimenAuto > 0:
		return patterns.Auto
	case m.dimen.flags&dimenAbsolute > 0:
		return patterns.Just
	case m.dimen.flags&dimenInitial > 0:
		return patterns.Initial
	case m.dimen.flags&dimenInherit > 0:
		return patterns.Inherit
	}
	return patterns.Default
}

func (m *DMatchExpr[T]) With(du *dimen.DU) *DMatchExpr[T] {
	*du = m.dimen.d
	return m
}

func (m *DMatchExpr[T]) Const(x T) T {
	return x
}

// ---------------------------------------------------------------------------

// IsNone returns true if d is unset.
func (d DimenT) IsNone() bool {
	return d.flags == dimenUnset
}

// IsRelative returns true if d represents a valid relative dimension (`%`, `em`, etc.).
func (d DimenT) IsRelative() bool {
	return d.flags&relativeMask > 0
}

// IsPercent returns true if d represents a percentage dimension (`%`).
func (d DimenT) IsPercent() bool {
	return d.flags&dimenPercent > 0
}

// IsAbsolute returns true if d represents a valid absolute dimension.
func (d DimenT) IsAbsolute() bool {
	return d.flags == dimenAbsolute
}

// ---------------------------------------------------------------------------

var dimenPattern = regexp.MustCompile(`^([+\-]?[0-9]+)(%|[a-zA-Z]{2,4})?$`)

// ParseDimen parses a string to return an optional dimension. Syntax is CSS Unit.
// Valid dimensions are
//
//     15px
//     80%
//     -33rem
//
func ParseDimen(s string) (DimenT, error) {
	// tracer().Debugf("parse dimen string = '%s'", s)
	if s == "" || s == "none" {
		return DimenT{}, nil
	}
	switch s {
	case "thin":
		return JustDimen(dimen.PX / 2), nil
	case "medium":
		return JustDimen(dimen.PX), nil
	case "thick":
		return JustDimen(dimen.PX * 2), nil
	}
	d := dimenPattern.FindStringSubmatch(s)
	if len(d) < 2 {
		return DimenT{}, errors.New("format error parsing dimension")
	}
	scale := dimen.SP
	dim := JustDimen(0)
	if len(d) > 2 {
		switch d[2] {
		case "pt", "PT":
			scale = dimen.PT
		case "mm", "MM":
			scale = dimen.MM
		case "bp", "px", "BP", "PX":
			scale = dimen.BP
		case "cm", "CM":
			scale = dimen.CM
		case "in", "IN":
			scale = dimen.IN
		case "", "sp", "SP":
			scale = dimen.SP
		default:
			u := strings.ToLower(d[2])
			if unit, ok := relUnitStringMap[u]; ok {
				dim = DimenT{}
				dim.flags = unit
			} else {
				return DimenT{}, errors.New("format error parsing dimension")
			}
		}
	}
	n, err := strconv.Atoi(d[1])
	if err != nil { // this cannot happen
		return DimenT{}, errors.New("format error parsing dimension")
	}
	dim.d = dimen.DU(n) * scale
	return dim, nil
}

// UnitString returns 'sp' (scaled points) for non-relative dimensions and a string
// denoting the defined unit for relative dimensions.
func (o DimenT) UnitString() string {
	if o.IsRelative() {
		if unit, ok := relUnitMap[o.flags&relativeMask]; ok {
			return unit
		}
	}
	return "sp"
}

var relUnitMap map[uint32]string = map[uint32]string{
	dimenEM:      "em",
	dimenEX:      "ex",
	dimenCH:      "ch",
	dimenREM:     "rem",
	dimenVW:      "vw",
	dimenVH:      "vh",
	dimenVMIN:    "vmin",
	dimenVMAX:    "vmax",
	dimenPercent: "%",
}

var relUnitStringMap map[string]uint32 = map[string]uint32{
	"em":   dimenEM,
	"ex":   dimenEX,
	"ch":   dimenCH,
	"rem":  dimenREM,
	"vw":   dimenVW,
	"vh":   dimenVH,
	"vmin": dimenVMIN,
	"vmax": dimenVMAX,
	`%`:    dimenPercent,
}
