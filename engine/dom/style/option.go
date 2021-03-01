package style

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/option"
)

// PropertyType is a helper type for special values of properties, e.g.:
//
//     auto
//     initial
//     inherit
//
type PropertyType int

// Auto, Inherit and Initial are constant values for options-matching.
// Use with
//     option.Of{
//          style.Auto: …   // will match a DimenT with value "auto"
//     }
const (
	Auto       PropertyType = 1 // for option matching
	Inherit    PropertyType = 2 // for option matching
	Initial    PropertyType = 3 // for option matching
	FontScaled PropertyType = 4 // for option matching: dimension is font-dependent
)

const (
	dimenNone uint32 = 0

	dimenAbsolute uint32 = 0x0001
	dimenAuto     uint32 = 0x0002
	dimenInherit  uint32 = 0x0004
	dimenInitial  uint32 = 0x0008

	dimenEM    uint32 = 0x0010
	dimenEX    uint32 = 0x0020
	dimenCH    uint32 = 0x0040
	dimenREM   uint32 = 0x0080
	dimenVW    uint32 = 0x0100
	dimenVH    uint32 = 0x0200
	dimenVMIN  uint32 = 0x0400
	dimenVMAX  uint32 = 0x0800
	dimenPRCNT uint32 = 0x1000
)

// --- DimenT-----------------------------------------------------------------

// DimenT is an option type for CSS dimensions.
type DimenT struct {
	d     dimen.Dimen
	flags uint32
}

// SomeDimen creates an optional dimen with an initial value of x.
func SomeDimen(x dimen.Dimen) DimenT {
	return DimenT{d: x, flags: dimenAbsolute}
}

// Dimen creates an optional dimen without an initial value.
func Dimen() DimenT {
	return DimenT{d: 0, flags: dimenNone}
}

// Match is part of interface option.Type.
func (o DimenT) Match(choices interface{}) (value interface{}, err error) {
	return option.Match(o, choices)
}

// Equals is part of interface option.Type.
func (o DimenT) Equals(other interface{}) bool {
	T().Debugf("Dimen EQUALS %v ? %v", o, other)
	switch i := other.(type) {
	case dimen.Dimen:
		return o.Unwrap() == i
	case int32:
		return o.Unwrap() == dimen.Dimen(i)
	case int:
		return o.Unwrap() == dimen.Dimen(i)
	case PropertyType:
		switch i {
		case Auto:
			return o.flags&dimenAuto > 0
		case Initial:
			return o.flags&dimenInitial > 0
		case Inherit:
			return o.flags&dimenInherit > 0
		case FontScaled:
			return o.flags&dimenEM > 0 || o.flags&dimenEX > 0 ||
				o.flags&dimenREM > 0 || o.flags&dimenCH > 0
		}
	case string:
		switch i {
		case "%":
			return o.IsRelative()
		}
	}
	return false
}

// Unwrap returns the underlying dimension of o.
func (o DimenT) Unwrap() dimen.Dimen {
	return o.d
}

// IsNone returns true if o is unset.
func (o DimenT) IsNone() bool {
	return o.flags == dimenNone
}

// IsRelative returns true if o represents a valid relative dimension (`%`, `em`, etc.).
func (o DimenT) IsRelative() bool {
	return o.flags&0xfff0 > 0
}

func (o DimenT) String() string {
	if o.IsNone() {
		return "DimenT.None"
	}
	switch o.flags & 0x000f {
	case dimenAuto:
		return "auto"
	case dimenInitial:
		return "initial"
	case dimenInherit:
		return "inherit"
	}
	if o.IsRelative() {
		if unit, ok := relUnitMap[o.flags&0xfff0]; ok {
			return fmt.Sprintf("%d%s", o.d, unit)
		}
	}
	return fmt.Sprintf("%dsp", o.d)
}

var relUnitMap map[uint32]string = map[uint32]string{
	dimenEM:    "em",
	dimenEX:    "ex",
	dimenCH:    "ch",
	dimenREM:   "rem",
	dimenVW:    "vw",
	dimenVH:    "vh",
	dimenVMIN:  "vmin",
	dimenVMAX:  "vmax",
	dimenPRCNT: "%",
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
	"%":    dimenPRCNT,
}

// DimenOption returns an optional dimension type from a property string.
// It will never return an error, even with illegal input, but instead will then
// return an unset dimension.
func (p Property) DimenOption() DimenT {
	switch p {
	case "auto":
		return DimenT{flags: dimenAuto}
	case "initial":
		return DimenT{flags: dimenInitial}
	case "inherit":
		return DimenT{flags: dimenInherit}
	}
	d, err := ParseDimen(string(p))
	if err != nil {
		return Dimen()
	}
	return d
}

var dimenPattern = regexp.MustCompile(`^([+\-]?[0-9]+)(%|[A-Z]{2,4})?$`)

// ParseDimen parses a string to return an optional dimension. Syntax is CSS Unit.
// Valid dimensions are
//
//     15px
//     80%
//     -33rem
//
func ParseDimen(s string) (DimenT, error) {
	d := dimenPattern.FindStringSubmatch(s)
	if len(d) < 2 {
		return Dimen(), errors.New("format error parsing dimension")
	}
	scale := dimen.SP
	dim := DimenT{}
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
			if unit, ok := relUnitStringMap[d[2]]; ok {
				dim.flags = unit
			} else {
				return Dimen(), errors.New("format error parsing dimension")
			}
		}
	}
	n, err := strconv.Atoi(d[1])
	if err != nil { // this cannot happen
		return Dimen(), errors.New("format error parsing dimension")
	}
	dim.d = dimen.Dimen(n) * scale
	return dim, nil
}
