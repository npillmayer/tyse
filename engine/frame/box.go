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
	"github.com/npillmayer/tyse/core/option"
	"github.com/npillmayer/tyse/engine/dom/style/css"
)

type Rect struct {
	TopL dimen.Point
	W    css.DimenT
	H    css.DimenT
}

type Size struct {
	W css.DimenT
	H css.DimenT
}

// Box type, following the CSS box model.
type Box struct {
	Rect            // either content box or border box, depending on box-sizing
	Min             Size
	Max             Size
	BorderBoxSizing bool          // box-sizing = border-box ?
	Padding         [4]css.DimenT // inside of border
	BorderWidth     [4]css.DimenT // thickness of border
	Margins         [4]css.DimenT // outside of border, maybe unknown
}

// For padding, margins, etc. 4-way values always start at the top and travel
// clockwise.
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

// --- Handling of box dimensions --------------------------------------------

// ContentWidth returns the width of the content box.
// If this box has box-sizing set to `border-box` and the width dimensions do
// not have fixed values, an unset dimension is returned.
func (box *Box) ContentWidth() css.DimenT {
	if !box.BorderBoxSizing {
		return box.W
	}
	if box.HasFixedBorderBoxWidth(false) {
		w := box.W.Unwrap()
		w -= box.Padding[Left].Unwrap()
		w -= box.Padding[Right].Unwrap()
		w -= box.BorderWidth[Left].Unwrap()
		w -= box.BorderWidth[Right].Unwrap()
		return css.SomeDimen(w)
	}
	return css.Dimen()
}

// FixContentWidth sets a known value for the width of the content box.
// If padding or border have any %-relative values, those will be set to fixed
// dimensions as well.
// If box has box-sizing set to `border-box` and one of the width dimensions is
// of unknown value, false is returned and the content width is not set.
func (box *Box) FixContentWidth(w dimen.Dimen) bool {
	box.FixPaddingAndBorderWidth(w)
	//box.FixBorderBoxPaddingAndBorderWidth(w)
	if box.BorderBoxSizing {
		if !box.Padding[Left].IsAbsolute() || !box.Padding[Right].IsAbsolute() ||
			!box.BorderWidth[Left].IsAbsolute() || !box.BorderWidth[Right].IsAbsolute() {
			return false
		}
		w += box.Padding[Left].Unwrap()
		w += box.Padding[Right].Unwrap()
		w += box.BorderWidth[Left].Unwrap()
		w += box.BorderWidth[Right].Unwrap()
	}
	box.W = css.SomeDimen(w)
	return true
}

// HasFixedBorderBoxWidth return true if box.W, horizontal margins and border width for
// left and right border have fixed (known) values.
// If includeMargins is true, left and right margins are checked as well.
func (box *Box) HasFixedBorderBoxWidth(includeMargins bool) bool {
	if includeMargins {
		if !box.Margins[Left].IsAbsolute() || !box.Margins[Right].IsAbsolute() {
			return false
		}
	}
	if !box.Padding[Left].IsAbsolute() || !box.Padding[Right].IsAbsolute() ||
		!box.BorderWidth[Left].IsAbsolute() || !box.BorderWidth[Right].IsAbsolute() ||
		!box.W.IsAbsolute() {
		return false
	}
	return true
}

// BorderBoxWidth returns the width of a box, including padding and border.
// If box has box-sizing set to `content-box`and at least one of the dimensions
// is not of fixed value, an unset dimension is returned.
func (box *Box) BorderBoxWidth() css.DimenT {
	if box.BorderBoxSizing {
		return box.W
	}
	if box.HasFixedBorderBoxWidth(false) {
		w := box.W.Unwrap()
		w += box.Padding[Left].Unwrap()
		w += box.Padding[Right].Unwrap()
		w += box.BorderWidth[Left].Unwrap()
		w += box.BorderWidth[Right].Unwrap()
		return css.SomeDimen(w)
	}
	return css.Dimen()
}

// FixBorderBoxWidth sets a known width for a box.
//
// If box has box-sizing set to `content-box` and at least one of the
// internal widths has a variable value, the size is not set.
// Otherwise padding and border have to be set beforehand to have a correct result
// for the width-calculation.
//
// Will return true if all inner horizontal dimensions (i.e., excluding
// margins) are fixed.
func (box *Box) FixBorderBoxWidth(w dimen.Dimen) bool {
	if box.BorderBoxSizing {
		box.W = css.SomeDimen(w)
		return box.FixBorderSizedBoxPaddingAndBorderWidth(w)
	}
	if !box.Padding[Left].IsAbsolute() || !box.Padding[Right].IsAbsolute() ||
		!box.BorderWidth[Left].IsAbsolute() || !box.BorderWidth[Right].IsAbsolute() {
		return false
	}
	w -= box.Padding[Left].Unwrap()
	w -= box.Padding[Right].Unwrap()
	w -= box.BorderWidth[Left].Unwrap()
	w -= box.BorderWidth[Right].Unwrap()
	box.W = css.SomeDimen(w)
	return true
}

// TotalWidth returns the overall width of a box, including margins.
// If one of the dimensions is not of fixed value, an unset dimension is returned.
func (box *Box) TotalWidth() css.DimenT {
	if box.HasFixedBorderBoxWidth(true) {
		w := box.BorderBoxWidth().Unwrap()
		w += box.Margins[Left].Unwrap()
		w += box.Margins[Right].Unwrap()
		return css.SomeDimen(w)
	}
	return css.Dimen()
}

// Height functions not ot yet implemented.
func (box *Box) TotalHeight() dimen.Dimen {
	panic("TODO")
	// return box.BotR.Y + box.Padding[Bottom] + box.BorderWidth[Bottom] + box.Margins[Bottom] -
	// 	box.TopL.Y - box.Padding[Top] - box.BorderWidth[Top] - box.Margins[Top]
}

// SetWidth sets the width of a box. Depending on wether `box-sizing` is
// set to `content-box` (default) or `border-box`, this box.W will then
// reflect either the content box width or the border box width.
func (box *Box) SetWidth(w css.DimenT) {
	box.W = w
}

// ---------------------------------------------------------------------------

// CollapseMargins returns the greater margin between bottom margin of box1 and
// top margin of box2, and the smaller one as the second return value.
//
// If any of the boxes' margins are unset, return values may be unset, too.
//
func CollapseMargins(box1, box2 *Box) (css.DimenT, css.DimenT) {
	if box1 == nil {
		if box2 == nil {
			return css.SomeDimen(0), css.SomeDimen(0)
		}
		return box2.Margins[Top], css.SomeDimen(0)
	} else if box2 == nil {
		return box1.Margins[Bottom], css.SomeDimen(0)
	}
	return css.MaxDimen(box1.Margins[Bottom], box2.Margins[Top]),
		css.MinDimen(box1.Margins[Bottom], box2.Margins[Top])
}

// FixPaddingAndBorderWidth fixes padding and boder width values of %-dimension
// if the content-width of box is fixed.
//
// Will return true if all 4 paddings have fixed dimensions.
//
// The padding size is relative to the width of that element’s content area
// (i.e. the width inside, and not including, the padding, border and margin of
// the element).
// So, if your <h1> is 500px wide, 10% padding = 0.1 × 500 pixels = 50 pixels.
// Note that top and bottom padding will also be 10% of the _width_ of the element,
// not 10% of the height of the element.
func (box *Box) FixPaddingAndBorderWidth(w dimen.Dimen) bool {
	fixed := true
	for dir := Top; dir <= Left; dir++ {
		if box.Padding[dir].UnitString() == "%" {
			p := box.Padding[dir].Unwrap()
			box.Padding[dir] = css.SomeDimen(p * w / 100)
		}
		if box.BorderWidth[dir].UnitString() == "%" {
			p := box.BorderWidth[dir].Unwrap()
			box.BorderWidth[dir] = css.SomeDimen(p * w / 100)
		}
		if !box.Padding[dir].IsAbsolute() || !box.BorderWidth[dir].IsAbsolute() {
			fixed = false
		}
	}
	return fixed
}

func (box *Box) FixBorderSizedBoxPaddingAndBorderWidth(w dimen.Dimen) bool {
	if !box.BorderBoxSizing {
		panic("content box sizing set, cannot fix border box")
	}
	pcnt, total := 100, int(w)
	addPcnt := func(p interface{}) interface{} {
		pcnt += p.(int)
		return 0
	}
	subtrTotal := func(t interface{}) interface{} {
		total -= t.(int)
		return 0
	}
	x := box.Padding[Left]
	if _, err := x.Match(option.Of{
		css.FixedValue: subtrTotal, // total = total - x
		"%":            addPcnt,    // pcnt = pcnt + x
	}); err != nil { // for other than fixed and %
		return false
	}
	x = box.Padding[Right]
	if _, err := x.Match(option.Of{
		css.FixedValue: subtrTotal, // total = total - x
		"%":            addPcnt,    // pcnt = pcnt + x
	}); err != nil { // for other than fixed and %
		return false
	}
	x = box.BorderWidth[Left]
	if _, err := x.Match(option.Of{
		css.FixedValue: subtrTotal, // total = total - x
		"%":            addPcnt,    // pcnt = pcnt + x
	}); err != nil { // for other than fixed and %
		return false
	}
	x = box.BorderWidth[Right]
	if _, err := x.Match(option.Of{
		css.FixedValue: subtrTotal, // total = total - x
		"%":            addPcnt,    // pcnt = pcnt + x
	}); err != nil { // for other than fixed and %
		return false
	}
	unit := total / pcnt
	return box.FixPaddingAndBorderWidth(dimen.Dimen(unit * 100))
}
