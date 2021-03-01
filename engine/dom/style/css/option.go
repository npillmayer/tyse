package css

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/option"
	"github.com/npillmayer/tyse/engine/dom/style"
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
//          style.Auto: …   // will match a CSS property option-type with value "auto"
//     }
const (
	Auto          PropertyType = 1 // for option matching
	Inherit       PropertyType = 2 // for option matching
	Initial       PropertyType = 3 // for option matching
	FontScaled    PropertyType = 4 // for option matching: dimension is font-dependent
	ViewScaled    PropertyType = 5 // for option matching: dimension is viewport-dependent
	ContentScaled PropertyType = 6 // for option matching: dimension is content-dependent
)

const (
	dimenNone uint32 = 0

	dimenAbsolute uint32 = 0x0001
	dimenAuto     uint32 = 0x0002
	dimenInherit  uint32 = 0x0003
	dimenInitial  uint32 = 0x0004

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
	dimenPRCNT   uint32 = 0x0900
	relativeMask uint32 = 0x00f0
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
	case DimenT:
		return o.d == i.d && o.flags == i.flags
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
		case ViewScaled:
			return o.flags&dimenVW > 0 || o.flags&dimenVH > 0 ||
				o.flags&dimenVMIN > 0 || o.flags&dimenVMAX > 0
		case ContentScaled:
			return o.flags&contentMask > 0
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
	return o.flags&relativeMask > 0
}

// IsAbsolute returns true if o represents a valid absolute dimension.
func (o DimenT) IsAbsolute() bool {
	return o.flags == dimenAbsolute
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
		if unit, ok := relUnitMap[o.flags&relativeMask]; ok {
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
func DimenOption(p style.Property) DimenT {
	switch p {
	case style.NullStyle:
		return Dimen()
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

// MaxDimen returns the greater of two dimensions.
func MaxDimen(d1, d2 DimenT) DimenT {
	max, _ := d1.Match(option.Maybe{
		option.None: d2,
		option.Some: option.Safe(d2.Match(option.Maybe{
			option.None: d1,
			option.Some: dimen.Max(d1.Unwrap(), d2.Unwrap()),
		})),
	})
	return max.(DimenT)
}

// MinDimen returns the lesser of two dimensions.
func MinDimen(d1, d2 DimenT) DimenT {
	max, _ := d1.Match(option.Maybe{
		option.None: d2,
		option.Some: option.Safe(d2.Match(option.Maybe{
			option.None: d1,
			option.Some: dimen.Min(d1.Unwrap(), d2.Unwrap()),
		})),
	})
	return max.(DimenT)
}

// --- PositionT -------------------------------------------------------------

// Position is an enum type for the CSS position property.
type Position uint16

// Enum values for type Position
const (
	PositionUnknown  Position = iota
	PositionStatic            // CSS static (default)
	PositionRelative          // CSS relative
	PositionAbsolute          // CSS absolute
	PositionFixed             // CSS fixed
	PositionSticky            // CSS sticky, currently mapped to relative
)

// PositionT is an option type for CSS positions.
type PositionT struct {
	p       Position
	Offsets []DimenT
}

// SomePosition creates an optional position with an initial value of x.
func SomePosition(x Position) PositionT {
	return PositionT{p: x}
}

// Match is part of interface option.Type.
func (o PositionT) Match(choices interface{}) (value interface{}, err error) {
	return option.Match(o, choices)
}

// Equals is part of interface option.Type.
func (o PositionT) Equals(other interface{}) bool {
	T().Debugf("Position EQUALS %v ? %v", o, other)
	switch p := other.(type) {
	case Position:
		return o.Unwrap() == p
	case string:
		if pp, ok := positionStringMap[p]; ok {
			return o.p == pp
		}
	}
	return false
}

// Unwrap returns the underlying position of o.
func (o PositionT) Unwrap() Position {
	return o.p
}

// IsNone returns true if o is unset.
func (o PositionT) IsNone() bool {
	return o.p == PositionUnknown
}

func (o PositionT) String() string {
	if o.IsNone() {
		return "PositionT.None"
	}
	if p, ok := positionMap[o.p]; ok {
		return p
	}
	return "PositionT.None"
}

var positionMap map[Position]string = map[Position]string{
	PositionStatic:   "static",
	PositionRelative: "relative",
	PositionAbsolute: "absolute",
	PositionFixed:    "fixed",
	PositionSticky:   "sticky",
}

var positionStringMap map[string]Position = map[string]Position{
	"static":   PositionStatic,
	"relative": PositionRelative,
	"absolute": PositionAbsolute,
	"fixed":    PositionFixed,
	"sticky":   PositionSticky,
}

// ParsePosition parses a string and returns an option-type for positions.
// It will never return an error, but rather an unset position in case of illegal input.
func ParsePosition(s string) PositionT {
	if p, ok := positionStringMap[s]; ok {
		return SomePosition(p)
	}
	return PositionT{}
}

// PositionOption returns an optional position type from properties.
// Properties `top`, `right`, `bottom` and `left` will be made accessable as option types,
// if appropriate.
//
// Will never return an error, even with illegal input, but instead will then
// return an unset position.
//
func PositionOption(styler style.Styler) PositionT {
	pos := GetLocalProperty(styler.Styles(), "position")
	if pos == style.NullStyle {
		return PositionT{}
	}
	p := ParsePosition(string(pos))
	if !p.IsNone() && p.Unwrap() != PositionStatic {
		p.Offsets = make([]DimenT, 4)
		p.Offsets[0] = DimenOption(GetLocalProperty(styler.Styles(), "top"))
		p.Offsets[1] = DimenOption(GetLocalProperty(styler.Styles(), "right"))
		p.Offsets[2] = DimenOption(GetLocalProperty(styler.Styles(), "bottom"))
		p.Offsets[3] = DimenOption(GetLocalProperty(styler.Styles(), "left"))
	}
	return p
}
