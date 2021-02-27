package frame

/*
BSD License

Copyright (c) 2017–2021, Norbert Pillmayer

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
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"image/color"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/font"
)

// Box type, following the CSS box model.
type Box struct {
	dimen.Rect
	Min         dimen.Point
	Max         dimen.Point
	Padding     [4]dimen.Dimen // inside of border
	BorderWidth [4]dimen.Dimen // thickness of border
	Margins     [4]dimen.Dimen // outside of border
}

// For padding, margins
const (
	Top int = iota
	Right
	Bottom
	Left
)

// Box styling: We follow the CSS paradigm for boxes. Boxes are stylable
// objects which have dimensions, spacing, borders and colors.
//
// Some boxes may just implement a subset of the styling parameters. Most
// notably this holds for glyphs: Glyphs may have styled their content only.
// No border or additional spacing is possible with glyphs.

// ColorStyle is a type for styling with color.
type ColorStyle struct {
	Foreground color.Color
	Background color.Color // may be (semi-)transparent
}

// TextStyle is a type for styling text.
type TextStyle struct {
	Typecase *font.TypeCase
}

// BorderStyle is a type for simple borders.
type BorderStyle struct {
	LineColor    color.Color
	LineStyle    int8
	CornerRadius dimen.Dimen
}

// LineStyle is a type for border line styles.
type LineStyle int8

// We support these line styles only
const (
	LSSolid  LineStyle = 0
	LSDashed LineStyle = 1
	LSDotted LineStyle = 2
)

// Styling rolls all styling options into one type.
type Styling struct {
	TextStyle TextStyle
	Colors    ColorStyle
	Border    BorderStyle
}

// StyledBox is a type for a fully stylable box.
type StyledBox struct {
	Box
	Styling *Styling
}

/*

// Glyph is a box for glyphs. Glyphs currently are content-stylable only (no borders).
//
// Wikipedia: In typography, a glyph [...] is an elemental symbol within
// an agreed set of symbols, intended to represent a readable character
// for the purposes of writing. [ Copyright (c) Wikipedia.com, 2017 ]
type Glyph struct {
	TextStyle *TextStyle
	Colors    *ColorStyle
	CharPos   rune
}
*/

// Normalize sorts the corner coordinates into correct order.
func (box *Box) Normalize() *Box {
	if box.TopL.X > box.BotR.X {
		box.TopL.X, box.BotR.X = box.BotR.X, box.TopL.X
	}
	if box.TopL.Y > box.BotR.Y {
		box.TopL.Y, box.BotR.Y = box.BotR.Y, box.TopL.Y
	}
	return box
}

func (box *Box) InnerWidth() dimen.Dimen {
	return box.BotR.X - box.TopL.X
}

func (box *Box) FullWidth() dimen.Dimen {
	return box.BotR.X + box.Padding[Right] + box.BorderWidth[Right] + box.Margins[Right] -
		box.TopL.X - box.Padding[Left] - box.BorderWidth[Left] - box.Margins[Left]
}

func (box *Box) FullHeight() dimen.Dimen {
	return box.BotR.Y + box.Padding[Bottom] + box.BorderWidth[Bottom] + box.Margins[Bottom] -
		box.TopL.Y - box.Padding[Top] - box.BorderWidth[Top] - box.Margins[Top]
}

// Shift a box along a vector. The size of the box is unchanged.
func (box *Box) Shift(vector dimen.Point) *Box {
	box.TopL.Shift(vector)
	box.BotR.Shift(vector)
	return box
}

// Enlarge a box in x- and y-direction. For shrinking, use negative
// argument(s).
func (box *Box) Enlarge(scales dimen.Point) *Box {
	box.BotR.X = box.BotR.X + scales.X
	box.BotR.Y = box.BotR.Y + scales.Y
	return box
}

// CollapseMargins returns the greater margin between bottom margin of box1 and
// top margin of box2, and the smaller one as the second return value.
func CollapseMargins(box1, box2 *Box) (dimen.Dimen, dimen.Dimen) {
	if box1 == nil {
		if box2 == nil {
			return 0, 0
		}
		return box2.Margins[Top], 0
	} else if box2 == nil {
		return box1.Margins[Bottom], 0
	}
	return dimen.Max(box1.Margins[Bottom], box2.Margins[Top]),
		dimen.Min(box1.Margins[Bottom], box2.Margins[Top])
}

/*
// Method for boxing content into a horizontal box. Content is given as a
// node list. The nodes will be enclosed into a new box.
// The box may be set to a target size.
// Parameters for styling class and/or identifier may be provided.
func HBoxKhipu(nl *Khipu, target p.Dimen, identifier string, class string) *TypesetBox {
	box := &TypesetBox{}
	box.Cord = nl
	box.Style.StylingIdentifier = identifier
	box.Style.StylingClass = class
	box.Width = target
	_, max, min := nl.Measure(0, -1)
	if min > target {
		fmt.Println("overfull hbox")
	} else if max < target {
		fmt.Println("underfull hbox")
	}
	box.Height, box.Depth = nl.MaxHeightAndDepth(0, -1)
	return box
}

// --- Boxes as khipu knots --------------------------------------------------

// KTBox is a khipu knot type
const KTBox = khipu.KTUserDefined + 1

func (box *Box) Type() khipu.KnotType {
	return KTBox
}

func (box *Box) W() dimen.Dimen {
	return box.Width()
}

func (box *Box) MinW() dimen.Dimen {
	return box.Width()
}

func (box *Box) MaxW() dimen.Dimen {
	return box.Width()
}

func (box *Box) IsDiscardable() bool {
	return false
}
*/
