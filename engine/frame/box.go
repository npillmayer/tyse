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
	"errors"
	"fmt"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/option"
	"github.com/npillmayer/tyse/engine/dom/style/css"
)

type Rect struct {
	TopL dimen.Point
	Size
	// W css.DimenT
	// H css.DimenT
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

// StyledBox is a type for a fully stylable box.
type StyledBox struct {
	Box
	Styles *Styling
}

// --- Handling of box dimensions --------------------------------------------

// DebugString returns a textual representation of a box's dimensions.
// Intended for debugging.
func (box *Box) DebugString() string {
	s := fmt.Sprintf("box{\n   w=%v, h=%v  (bbox-sz=%v)\n", box.W, box.H, box.BorderBoxSizing)
	s += fmt.Sprintf("   p.top=%v, p.right=%v, p.bottom=%v, p.left=%v\n",
		box.Padding[Top], box.Padding[Right],
		box.Padding[Bottom], box.Padding[Left])
	s += fmt.Sprintf("   b.top=%v, b.right=%v, b.bottom=%v, b.left=%v\n",
		box.BorderWidth[Top], box.BorderWidth[Right],
		box.BorderWidth[Bottom], box.BorderWidth[Left])
	s += fmt.Sprintf("   m.top=%v, m.right=%v, m.bottom=%v, m.left=%v\n",
		box.Margins[Top], box.Margins[Right],
		box.Margins[Bottom], box.Margins[Left])
	s += "}"
	return s
}

// SetWidth sets the width of a box. Depending on wether `box-sizing` is
// set to `content-box` (default) or `border-box`, this box.W will then
// reflect either the content box width or the border box width.
//
// TODO remove this ?
func (box *Box) SetWidth(w css.DimenT) {
	box.W = w
}

// ContentWidth returns the width of the content box.
// If this box has box-sizing set to `border-box` and the width dimensions do
// not have fixed values, an unset dimension is returned.
func (box *Box) ContentWidth() css.DimenT {
	if !box.BorderBoxSizing {
		return box.W
	}
	if box.HasFixedBorderBoxWidth(false) {
		w := box.W.Unwrap()
		w -= innerDecorationWidth(box).Unwrap()
		// w -= box.Padding[Left].Unwrap()
		// w -= box.Padding[Right].Unwrap()
		// w -= box.BorderWidth[Left].Unwrap()
		// w -= box.BorderWidth[Right].Unwrap()
		return css.SomeDimen(w)
	}
	return css.Dimen()
}

// ContentHeight returns the height of the content box.
// If this box has box-sizing set to `border-box` and the height dimensions do
// not have fixed values, an unset dimension is returned.
func (box *Box) ContentHeight() css.DimenT {
	if !box.BorderBoxSizing {
		return box.H
	}
	if box.HasFixedBorderBoxHeight(false) {
		h := box.H.Unwrap()
		h -= innerDecorationHeight(box).Unwrap()
		return css.SomeDimen(h)
	}
	return css.Dimen()
}

// FixContentWidth sets a known value for the width of the content box.
// If padding or border have any %-relative values, those will be set to fixed
// dimensions as well.
// If box has box-sizing set to `border-box` and one of the width dimensions is
// of unknown value, false is returned and the content width is not set.
func (box *Box) FixContentWidth(w dimen.Dimen) bool {
	W, ok := fixPaddingAndBorderWidthFromContentWidth(box, w)
	if !ok {
		return false
	}
	/*
		if box.BorderBoxSizing {
			decW := innerDecorationWidth(box)
			if decW.IsNone() {
				return false
			}
			//w += decW.Unwrap()
			//
			// if !box.Padding[Left].IsAbsolute() || !box.Padding[Right].IsAbsolute() ||
			// 	!box.BorderWidth[Left].IsAbsolute() || !box.BorderWidth[Right].IsAbsolute() {
			// 	return false
			// }
			// w += box.Padding[Left].Unwrap()
			// w += box.Padding[Right].Unwrap()
			// w += box.BorderWidth[Left].Unwrap()
			// w += box.BorderWidth[Right].Unwrap()
		}
	*/
	box.W = W
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

// HasFixedBorderBoxHeight return true if box.W, horizontal margins and border width for
// left and right border have fixed (known) values.
// If includeMargins is true, left and right margins are checked as well.
func (box *Box) HasFixedBorderBoxHeight(includeMargins bool) bool {
	//T().Debugf("fixed border box height ? => %s", box.DebugString())
	if includeMargins {
		if !box.Margins[Top].IsAbsolute() || !box.Margins[Bottom].IsAbsolute() {
			return false
		}
	}
	if !box.Padding[Top].IsAbsolute() || !box.Padding[Bottom].IsAbsolute() ||
		!box.BorderWidth[Top].IsAbsolute() || !box.BorderWidth[Bottom].IsAbsolute() ||
		!box.H.IsAbsolute() {
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
		w += innerDecorationWidth(box).Unwrap()
		// w += box.Padding[Left].Unwrap()
		// w += box.Padding[Right].Unwrap()
		// w += box.BorderWidth[Left].Unwrap()
		// w += box.BorderWidth[Right].Unwrap()
		return css.SomeDimen(w)
	}
	return css.Dimen()
}

// BorderBoxHeight returns the width of a box, including padding and border.
// If box has box-sizing set to `content-box`and at least one of the dimensions
// is not of fixed value, an unset dimension is returned.
func (box *Box) BorderBoxHeight() css.DimenT {
	if box.BorderBoxSizing {
		return box.H
	}
	if box.HasFixedBorderBoxHeight(false) {
		h := box.W.Unwrap()
		h += innerDecorationHeight(box).Unwrap()
		return css.SomeDimen(h)
	}
	return css.Dimen()
}

// FixBorderBoxWidth sets a known border box width for a box.
//
// If box has box-sizing set to `content-box` and at least one of the
// internal widths has a variable value, the size is not set.
// Otherwise padding and border have to be set beforehand to have a correct result
// for the width-calculation.
//
// Will return true if all inner horizontal dimensions (i.e., excluding
// margins) are fixed.
func (box *Box) FixBorderBoxWidth(w dimen.Dimen) {
	if box.BorderBoxSizing {
		box.W = css.SomeDimen(w)
		_, ok := fixPaddingAndBorderWidthFromBorderBoxWidth(box, w)
		if !ok {
			T().Errorf("cannot fix padding and border")
		}
		return
	}
	//contentW := box.fixPaddingAndBorderWidthFromBorderBox(w)
	T().Debugf("w = %v", w)
	contentW, ok := fixPaddingAndBorderWidthFromBorderBoxWidth(box, w)
	T().Debugf("contentW = %v", contentW)
	if !ok || contentW.IsNone() {
		T().Errorf("cannot fix padding and border")
		return
	}
	// decW := innerDecorationWidth(box)
	// if decW.IsNone() {
	// 	return false
	// }
	// w -= decW.Unwrap()
	// if !box.Padding[Left].IsAbsolute() || !box.Padding[Right].IsAbsolute() ||
	// 	!box.BorderWidth[Left].IsAbsolute() || !box.BorderWidth[Right].IsAbsolute() {
	// 	return false
	// }
	// w -= box.Padding[Left].Unwrap()
	// w -= box.Padding[Right].Unwrap()
	// w -= box.BorderWidth[Left].Unwrap()
	// w -= box.BorderWidth[Right].Unwrap()
	box.W = contentW
	return
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

// TotalHeight returns the overall height of a box.
func (box *Box) TotalHeight() css.DimenT {
	if box.HasFixedBorderBoxHeight(true) {
		h := box.BorderBoxHeight().Unwrap()
		h += box.Margins[Top].Unwrap()
		h += box.Margins[Bottom].Unwrap()
		return css.SomeDimen(h)
	}
	return css.Dimen()
}

func (box *Box) OuterBox() Rect {
	r := Rect{TopL: box.TopL}
	r.W = box.TotalWidth()
	r.H = box.TotalHeight()
	return r
}

// DecorationWidth returns the cumulated width of padding, borders and margins
// if all of them have known values, and an unset dimension otherwise.
func (box *Box) DecorationWidth(includeMargins bool) css.DimenT {
	w := dimen.Zero
	if includeMargins {
		if !box.Margins[Left].IsAbsolute() || !box.Margins[Right].IsAbsolute() {
			return css.Dimen()
		}
		w += box.Margins[Left].Unwrap()
		w += box.Margins[Right].Unwrap()
	}
	decW := innerDecorationWidth(box)
	if decW.IsNone() {
		return decW
	}
	return css.SomeDimen(w + decW.Unwrap())
	// if !box.Padding[Left].IsAbsolute() || !box.Padding[Right].IsAbsolute() ||
	// 	!box.BorderWidth[Left].IsAbsolute() || !box.BorderWidth[Right].IsAbsolute() {
	// 	return css.Dimen()
	// }
	// w += box.Padding[Left].Unwrap()
	// w += box.Padding[Right].Unwrap()
	// w += box.BorderWidth[Left].Unwrap()
	// w += box.BorderWidth[Right].Unwrap()
	// return css.SomeDimen(w)
}

func (box *Box) FixPercentages(enclosingWidth dimen.Dimen) bool {
	fixed := true
	for dir := Top; dir <= Left; dir++ {
		if box.Padding[dir].IsPercent() {
			p := box.Padding[dir].Unwrap()
			box.Padding[dir] = css.SomeDimen(p * enclosingWidth / 100)
		}
		if box.BorderWidth[dir].IsPercent() {
			p := box.BorderWidth[dir].Unwrap()
			box.BorderWidth[dir] = css.SomeDimen(p * enclosingWidth / 100)
		}
		if box.Margins[dir].IsPercent() {
			p := box.BorderWidth[dir].Unwrap()
			box.BorderWidth[dir] = css.SomeDimen(p * enclosingWidth / 100)
		}
		if !box.Padding[dir].IsAbsolute() || !box.BorderWidth[dir].IsAbsolute() ||
			!box.BorderWidth[dir].IsAbsolute() {
			fixed = false
		}
	}
	return fixed
}

// ----------------------------------------------------------------------------------

// InitEmptyBox initializes padding, border and margins to 0 and box.W to auto.
func InitEmptyBox(box *Box) *Box {
	if box == nil {
		box = &Box{}
	}
	box.Padding[Top] = css.SomeDimen(0)
	box.Padding[Right] = css.SomeDimen(0)
	box.Padding[Bottom] = css.SomeDimen(0)
	box.Padding[Left] = css.SomeDimen(0)
	box.BorderWidth[Top] = css.SomeDimen(0)
	box.BorderWidth[Right] = css.SomeDimen(0)
	box.BorderWidth[Bottom] = css.SomeDimen(0)
	box.BorderWidth[Left] = css.SomeDimen(0)
	box.Margins[Top] = css.SomeDimen(0)
	box.Margins[Right] = css.SomeDimen(0)
	box.Margins[Bottom] = css.SomeDimen(0)
	box.Margins[Left] = css.SomeDimen(0)
	//
	box.W = css.DimenOption("auto")
	return box
}

func innerDecorationWidth(box *Box) css.DimenT {
	if !box.Padding[Left].IsAbsolute() || !box.Padding[Right].IsAbsolute() ||
		!box.BorderWidth[Left].IsAbsolute() || !box.BorderWidth[Right].IsAbsolute() {
		return css.Dimen()
	}
	w := dimen.Zero
	w += box.Padding[Left].Unwrap()
	w += box.Padding[Right].Unwrap()
	w += box.BorderWidth[Left].Unwrap()
	w += box.BorderWidth[Right].Unwrap()
	return css.SomeDimen(w)
}

func innerDecorationHeight(box *Box) css.DimenT {
	if !box.Padding[Top].IsAbsolute() || !box.Padding[Bottom].IsAbsolute() ||
		!box.BorderWidth[Top].IsAbsolute() || !box.BorderWidth[Bottom].IsAbsolute() {
		return css.Dimen()
	}
	h := dimen.Zero
	h += box.Padding[Top].Unwrap()
	h += box.Padding[Bottom].Unwrap()
	h += box.BorderWidth[Top].Unwrap()
	h += box.BorderWidth[Bottom].Unwrap()
	return css.SomeDimen(h)
}

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

// FixPaddingAndBorderWidth fixes padding and border width values of %-dimension
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
func (box *Box) fixPaddingAndBorderWidth(w dimen.Dimen) bool {
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

func setWFromEnclosing(box *Box, enclw dimen.Dimen) {
	if !box.W.IsPercent() {
		return
	}
	box.W = css.SomeDimen(box.W.Unwrap() * enclw)
}
func fixPaddingAndBorderWidthFromBorderBoxWidth(box *Box, w dimen.Dimen) (css.DimenT, bool) {
	T().Debugf("fix padding from bbox")
	var hundredPcntW int64
	var W css.DimenT
	if !box.BorderBoxSizing {
		pcnt, total := int64(100), w
		for dir := Right; dir <= Left; dir += 2 { // horizontal
			if box.Padding[dir].IsAbsolute() {
				total -= box.Padding[dir].Unwrap()
			} else if box.Padding[dir].IsPercent() {
				pcnt += int64(box.Padding[dir].Unwrap())
			} else {
				return css.Dimen(), false
			}
			if box.BorderWidth[dir].IsAbsolute() {
				total -= box.BorderWidth[dir].Unwrap()
			} else if box.BorderWidth[dir].IsPercent() {
				pcnt += int64(box.BorderWidth[dir].Unwrap())
			} else {
				return css.Dimen(), false
			}
		}
		hundredPcntW = int64(total) * 100 / pcnt
		T().Debugf("100%% = %v", hundredPcntW)
		W = css.SomeDimen(dimen.Dimen(hundredPcntW))
	} else {
		hundredPcntW = int64(w)
		W = css.SomeDimen(w)
	}
	setPcntPaddingAndBorder(box, hundredPcntW)
	return W, true
}

func fixPaddingAndBorderWidthFromContentWidth(box *Box, w dimen.Dimen) (css.DimenT, bool) {
	var hundredPcntW int64
	var W css.DimenT
	if box.BorderBoxSizing {
		pcnt, total := int64(100), w
		for dir := Right; dir <= Left; dir += 2 { // horizontal
			if box.Padding[dir].IsAbsolute() {
				total += box.Padding[dir].Unwrap()
			} else if box.Padding[dir].IsPercent() {
				pcnt -= int64(box.Padding[dir].Unwrap())
			} else {
				return css.Dimen(), false
			}
			if box.BorderWidth[dir].IsAbsolute() {
				total += box.BorderWidth[dir].Unwrap()
			} else if box.BorderWidth[dir].IsPercent() {
				pcnt -= int64(box.BorderWidth[dir].Unwrap())
			} else {
				return css.Dimen(), false
			}
		}
		hundredPcntW = int64(total) * 100 / pcnt
		W = css.SomeDimen(dimen.Dimen(hundredPcntW))
	} else {
		hundredPcntW = int64(w)
		W = css.SomeDimen(w)
	}
	setPcntPaddingAndBorder(box, hundredPcntW)
	return W, true
}

func setPcntPaddingAndBorder(box *Box, hundredPcntW int64) {
	for dir := Top; dir <= Left; dir++ {
		if box.Padding[dir].IsPercent() {
			p := box.Padding[dir].Unwrap()
			box.Padding[dir] = css.SomeDimen(dimen.Dimen(int64(p) * hundredPcntW / 100))
		}
		if box.BorderWidth[dir].IsPercent() {
			p := box.BorderWidth[dir].Unwrap()
			box.BorderWidth[dir] = css.SomeDimen(dimen.Dimen(int64(p) * hundredPcntW / 100))
		}
	}
}

func (box *Box) fixPaddingAndBorderWidthFromBorderBox(w dimen.Dimen) css.DimenT {
	// if !box.BorderBoxSizing {
	// 	panic("content box sizing set, cannot fix border box")
	// }
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
		return css.Dimen()
	}
	x = box.Padding[Right]
	if _, err := x.Match(option.Of{
		css.FixedValue: subtrTotal, // total = total - x
		"%":            addPcnt,    // pcnt = pcnt + x
	}); err != nil { // for other than fixed and %
		return css.Dimen()
	}
	x = box.BorderWidth[Left]
	if _, err := x.Match(option.Of{
		css.FixedValue: subtrTotal, // total = total - x
		"%":            addPcnt,    // pcnt = pcnt + x
	}); err != nil { // for other than fixed and %
		return css.Dimen()
	}
	x = box.BorderWidth[Right]
	if _, err := x.Match(option.Of{
		css.FixedValue: subtrTotal, // total = total - x
		"%":            addPcnt,    // pcnt = pcnt + x
	}); err != nil { // for other than fixed and %
		return css.Dimen()
	}
	unit := total / pcnt
	return css.SomeDimen(dimen.Dimen(unit * 100))
	//return box.FixPaddingAndBorderWidth(dimen.Dimen(unit * 100))
}

// --- API for constraint width solving --------------------------------------

// ErrUnfixedScaledUnit is returned if a dimension calculation encounters a
// dimension-specification which is dependent on view-size or font-size.
//
var ErrUnfixedScaledUnit error = errors.New("font/view dependent dimension is unfixed")

// ErrContentScaling is returned if a dimension calculation encounters a
// dimension-specification which is dependent on the box's content.
var ErrContentScaling error = errors.New("box scales with content")

// ErrUnderspecified is returned if a dimension calculation cannot be completed
// because the input values are underspecified.
var ErrUnderspecified error = errors.New("box width dimensions are underspecified")

// FixDimensionsFromEnclosingWidth calculates missing/auto dimensions from the
// width of the enclosing box.
//
// This will distribute space according to the equation (ref. CSS spec):
//
//     margin-left + border-width-left + padding-left + width +
//       padding-right + border-width-right + margin-right = width of containing block
//
// Returns a flag denoting whether there was enough information to specify each width
// dimension.
//
func FixDimensionsFromEnclosingWidth(box *Box, enclosingWidth dimen.Dimen) (bool, error) {
	T().Debugf("fix contraint dimensions, enclosing = %v", enclosingWidth)
	fixIllegalDimensionSpecifications(box)
	box.FixPercentages(enclosingWidth)
	if err := checkForUnresolvedDependentDimensions(box); err != nil {
		return false, err
	}
	calc, err := box.W.Match(option.Of{
		option.None: calcWidthAsRest, // defaults to `auto`
		css.Auto:    calcWidthAsRest,
		option.Some: takeWidth,
	})
	if err != nil {
		return false, err
	}
	solve := asCalcFn(calc)
	w, err := solve(box, enclosingWidth)
	if err != nil {
		return false, err
	} else if !w.IsAbsolute() {
		return false, ErrUnderspecified
	}
	box.W = w
	T().Debugf("dimensions calculated from enclosing width: %s", box.DebugString())
	// if !box.Padding[dir].IsAbsolute() || !box.BorderWidth[dir].IsAbsolute() ||
	// 	!box.BorderWidth[dir].IsAbsolute() {
	// 	fixed = false
	// }
	return true, nil
}

type calcFn func(box *Box, enclosing dimen.Dimen) (css.DimenT, error)

func asCalcFn(f interface{}) calcFn {
	return f.(func(box *Box, enclosing dimen.Dimen) (css.DimenT, error))
}

func takeWidth(box *Box, enclosing dimen.Dimen) (css.DimenT, error) {
	T().Debugf("calculating width: simply take is as is = %v", box.W)
	fixed := distributeHorizontalMarginSpace(box, enclosing)
	if !fixed {
		return box.W, ErrUnderspecified
	}
	return box.W, nil
}

// Spec: If 'width' is set to 'auto', any other 'auto' values become '0'
// and 'width' follows from the resulting equality.
func calcWidthAsRest(box *Box, enclosing dimen.Dimen) (css.DimenT, error) {
	//T().Debugf("calculate width as rest for box %s", box.DebugString())
	left, err := box.Margins[Left].MatchToDimen(option.Of{
		option.None: dimen.Zero,
		css.Auto:    dimen.Zero,
		option.Some: box.Margins[Left].Unwrap(),
	})
	if err != nil {
		return css.Dimen(), err
	}
	box.Margins[Left] = css.SomeDimen(left)
	right, err := box.Margins[Right].MatchToDimen(option.Of{
		option.None: dimen.Zero,
		css.Auto:    dimen.Zero,
		option.Some: box.Margins[Left].Unwrap(),
	})
	if err != nil {
		return css.Dimen(), err
	}
	box.Margins[Right] = css.SomeDimen(right)
	width := enclosing - left - right
	T().Debugf("w = %v", width)
	if !box.BorderBoxSizing {
		var d css.DimenT
		if d = innerDecorationWidth(box); d.IsNone() {
			return d, ErrUnderspecified // this cannot happen
		}
		width -= d.Unwrap()
	}
	r := css.SomeDimen(width)
	T().Debugf("calculate width as rest to w = %v", r)
	return r, nil
	//return css.SomeDimen(width), nil
}

// distributeHorizontalMarginSpace distributes space into left and right margins
// after the border-box has been fixed.
func distributeHorizontalMarginSpace(box *Box, enclosing dimen.Dimen) bool {
	if !box.HasFixedBorderBoxWidth(false) {
		return false
	}
	w := box.BorderBoxWidth().Unwrap()
	remaining := enclosing - w
	left, right := box.Margins[Left], box.Margins[Right]
	l, err := left.Match(option.Of{
		css.Auto: option.Safe(right.Match(option.Of{
			css.Auto:    remaining / 2,
			option.Some: remaining - right.Unwrap(),
		})),
	})
	if err != nil {
		T().Errorf("distribute h-margins: %s", err.Error())
		return false
	}
	r := remaining - l.(dimen.Dimen)
	box.Margins[Left] = css.SomeDimen(l.(dimen.Dimen))
	box.Margins[Right] = css.SomeDimen(r)
	return true
}

// checkForUnresolvedDependentDimensions will return an error for box dimensions
// which are dependent on view-size, font-size or content.
func checkForUnresolvedDependentDimensions(box *Box) error {
	for dir := Top; dir <= Left; dir++ {
		if _, err := box.Padding[dir].Match(option.Of{
			option.None:       nil, // defaults to `auto`
			css.FontScaled:    option.Fail(ErrUnfixedScaledUnit),
			css.ViewScaled:    option.Fail(ErrUnfixedScaledUnit),
			css.ContentScaled: option.Fail(ErrContentScaling),
			option.Some:       nil,
		}); err != nil {
			return err
		}
		if _, err := box.BorderWidth[dir].Match(option.Of{
			option.None:       nil, // defaults to `auto`
			css.FontScaled:    option.Fail(ErrUnfixedScaledUnit),
			css.ViewScaled:    option.Fail(ErrUnfixedScaledUnit),
			css.ContentScaled: option.Fail(ErrContentScaling),
			option.Some:       nil,
		}); err != nil {
			return err
		}
		if _, err := box.Margins[dir].Match(option.Of{
			option.None:       nil, // defaults to `auto`
			css.FontScaled:    option.Fail(ErrUnfixedScaledUnit),
			css.ViewScaled:    option.Fail(ErrUnfixedScaledUnit),
			css.ContentScaled: option.Fail(ErrContentScaling),
			option.Some:       nil,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Property   Default    Valid values           Purpose
// ---------+----------+----------------------+-----------------------------------
// padding    Varies     length or percentage 	Controls the size of the padding.
//                                              Negative values are not allowed.
//                                              Percentages refer to width of the
//                                              containing block.
//
// Similar for border width.
//
func fixIllegalDimensionSpecifications(box *Box) {
	for dir := Top; dir <= Left; dir++ {
		padd := box.Padding[dir]
		if padd.Equals(css.Auto) || (padd.IsAbsolute() && padd.Unwrap() < 0) {
			padd = css.SomeDimen(0)
		}
		if !padd.IsAbsolute() {
			padd = css.SomeDimen(0)
		}
		bord := box.BorderWidth[dir]
		if bord.Equals(css.Auto) || (bord.IsAbsolute() && bord.Unwrap() < 0) {
			bord = css.SomeDimen(0)
		}
		if !bord.IsAbsolute() {
			bord = css.SomeDimen(0)
		}
	}
}
