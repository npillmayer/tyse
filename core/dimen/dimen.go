// Package dimen implements dimensions and units.
//
/*
BSD License

Copyright (c) 2017–21, Norbert Pillmayer (norbert@pillmayer.com)

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of this software nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.  */
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

// Dimen is a dimension type.
// Values are in scaled big points (different from TeX).
type Dimen int32

// Some pre-defined dimensions
const (
	Zero Dimen = 0
	SP   Dimen = 1       // scaled point = BP / 65536
	BP   Dimen = 65536   // big point (PDF) = 1/72 inch
	PX   Dimen = 65536   // "pixels"
	PT   Dimen = 65291   // printers point 1/72.27 inch
	MM   Dimen = 185771  // millimeters
	CM   Dimen = 1857710 // centimeters
	IN   Dimen = 4718592 // inch
)

// Infinity is the largest possible dimension
const Infinity = math.MaxInt32

// Some very stretchable dimensions
const Fil Dimen = Infinity - 3
const Fill Dimen = Infinity - 2
const Filll Dimen = Infinity - 1

// Some common paper sizes
var DINA4 = Point{210 * MM, 297 * MM}
var DINA5 = Point{148 * MM, 210 * MM}
var USLetter = Point{216 * MM, 279 * MM}
var USLegal = Point{216 * MM, 357 * MM}

// Stringer implementation.
func (d Dimen) String() string {
	return fmt.Sprintf("%dsp", int32(d))
}

// Points returns a dimension in big (PDF) points.
func (d Dimen) Points() float64 {
	return float64(d) / float64(BP)
}

// Point is a point on a page.
//
// TODO see methods in https://golang.org/pkg/image/#Point
type Point struct {
	X, Y Dimen
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
func (r Rect) Width() Dimen {
	return r.BotR.X - r.TopL.X
}

// Height returns the height of a rectangle, i.e. the difference between y-coordinates
// of bottom-right and top-left corner.
func (r Rect) Height() Dimen {
	return r.BotR.Y - r.TopL.Y
}

// ---------------------------------------------------------------------------

var dimenPattern = regexp.MustCompile(`^([+\-]?[0-9]+)(%|[cminpxtc]{2})?$`)

// ParseDimen parses a string to return a dimension. Syntax is CSS Unit.
// If a percentage value is given (`80%`), the second return value will be true.
//
func ParseDimen(s string) (Dimen, bool, error) {
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
	return Dimen(n) * scale, ispcnt, nil
}

// ---------------------------------------------------------------------------

// Min returns the smaller of two dimensions.
func Min(a, b Dimen) Dimen {
	if a < b {
		return a
	}
	return b
}

// Max returns the greater of two dimensions.
func Max(a, b Dimen) Dimen {
	if a > b {
		return a
	}
	return b
}
