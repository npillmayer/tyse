/*
Package dimen implements dimensions and units.

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package dimen

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
)

// Online dimension conversion for print:
// http://www.unitconversion.org/unit_converter/typography-ex.html

// DU is a 'design unit' typ.
// Values are in scaled big points (different from TeX).
type DU int32

// Some pre-defined dimensions
const (
	Zero DU = 0
	SP   DU = 1       // scaled point = BP / 65536
	BP   DU = 65536   // big point (PDF) = 1/72 inch
	PX   DU = 65536   // "pixels"
	PT   DU = 65291   // printers point 1/72.27 inch
	MM   DU = 185771  // millimeters
	CM   DU = 1857710 // centimeters
	IN   DU = 4718592 // inch
)

// Infinity is the largest possible dimension
const Infinity = math.MaxInt32

// Some very stretchable dimensions
const Fil DU = Infinity - 3
const Fill DU = Infinity - 2
const Filll DU = Infinity - 1

// Some common paper sizes
var DINA4 = Point{210 * MM, 297 * MM}
var DINA5 = Point{148 * MM, 210 * MM}
var USLetter = Point{216 * MM, 279 * MM}
var USLegal = Point{216 * MM, 357 * MM}

// Stringer implementation.
func (d DU) String() string {
	return fmt.Sprintf("%dsp", int32(d))
}

// Points returns a dimension in big (PDF) points.
func (d DU) Points() float64 {
	return float64(d) / float64(BP)
}

// Point is a point on a page.
//
// TODO see methods in https://golang.org/pkg/image/#Point
type Point struct {
	X, Y DU
}

// Origin is origin
var Origin = Point{0, 0}

// Shift a point along a vector.
func (p *Point) Shift(vector Point) *Point {
	p.X += vector.X
	p.Y += vector.Y
	return p
}

// Rect is a rectangle (on a page).
type Rect struct {
	TopL, BotR Point
}

// Width returns the width of a rectangle, i.e. the difference between x-coordinates
// of bottom-right and top-left corner.
func (r Rect) Width() DU {
	return r.BotR.X - r.TopL.X
}

// Height returns the height of a rectangle, i.e. the difference between y-coordinates
// of bottom-right and top-left corner.
func (r Rect) Height() DU {
	return r.BotR.Y - r.TopL.Y
}

// ---------------------------------------------------------------------------

var dimenPattern = regexp.MustCompile(`^([+\-]?[0-9]+)(%|[cminpxtc]{2})?$`)

// ParseDimen parses a string to return a dimension. Syntax is CSS Unit.
// If a percentage value is given (`80%`), the second return value will be true.
//
func ParseDimen(s string) (DU, bool, error) {
	d := dimenPattern.FindStringSubmatch(s)
	if len(d) < 2 {
		return 0, false, errors.New("format error parsing dimension")
	}
	scale := SP
	ispcnt := false
	if len(d) > 2 {
		switch d[2] {
		case "pt", "PT":
			scale = PT
		case "mm", "MM":
			scale = MM
		case "bp", "px", "BP", "PX":
			scale = BP
		case "cm", "CM":
			scale = CM
		case "in", "IN":
			scale = IN
		case "sp", "SP", "":
			scale = SP
		case "%":
			scale, ispcnt = 1, true
		default:
			return 0, false, errors.New("format error parsing dimension")
		}
	}
	n, err := strconv.Atoi(d[1])
	if err != nil {
		return 0, false, errors.New("format error parsing dimension")
	}
	return DU(n) * scale, ispcnt, nil
}

// ---------------------------------------------------------------------------

// Min returns the smaller of two dimensions.
func Min(a, b DU) DU {
	if a < b {
		return a
	}
	return b
}

// Max returns the greater of two dimensions.
func Max(a, b DU) DU {
	if a > b {
		return a
	}
	return b
}
