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
	"github.com/npillmayer/tyse/engine/dom/style/css"
)

type Rect struct {
	TopL dimen.Point
	W    css.DimenT
	H    css.DimenT
}

// Box type, following the CSS box model.
type Box struct {
	Rect
	Min             dimen.Point
	Max             dimen.Point
	BoxSizingExtend bool           // box-sizing = border-box ?
	Padding         [4]dimen.Dimen // inside of border
	BorderWidth     [4]dimen.Dimen // thickness of border
	Margins         [4]css.DimenT  // outside of border, maybe unknown
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

func (box *Box) ContentWidth() css.DimenT {
	return box.W
}

func (box *Box) SetContentWidth(w dimen.Dimen, shrink bool) {
	if shrink {
		w -= box.Padding[Left]
		w -= box.Padding[Right]
		w -= box.BorderWidth[Left]
		w -= box.BorderWidth[Right]
	}
	box.W = css.SomeDimen(w)
}

func (box *Box) Width() css.DimenT {
	if box.W.IsAbsolute() {
		return css.SomeDimen(box.W.Unwrap() + box.Padding[Left] + box.Padding[Right] +
			box.BorderWidth[Left] + box.BorderWidth[Right])
	}
	return box.W
}

// SetWidth sets a known width for a box. If box.BoxSizingExtend is set,
// padding and border have to be set beforehand to have a correct result
// for the width-calculation.
func (box *Box) SetWidth(w dimen.Dimen) {
	if box.BoxSizingExtend {
		box.W = css.SomeDimen(w - box.Padding[Left] - box.Padding[Right] -
			box.BorderWidth[Left] - box.BorderWidth[Right])
	} else {
		box.W = css.SomeDimen(w)
	}
}

func (box *Box) FullWidth() css.DimenT {
	w := box.Width()
	if w.IsAbsolute() && box.Margins[Left].IsAbsolute() && box.Margins[Right].IsAbsolute() {
		full := w.Unwrap() + box.Margins[Left].Unwrap() + box.Margins[Right].Unwrap()
		return css.SomeDimen(full)
	}
	if w.IsAbsolute() {
		return css.Dimen()
	}
	return box.W
}

func (box *Box) FullHeight() dimen.Dimen {
	panic("TODO")
	// return box.BotR.Y + box.Padding[Bottom] + box.BorderWidth[Bottom] + box.Margins[Bottom] -
	// 	box.TopL.Y - box.Padding[Top] - box.BorderWidth[Top] - box.Margins[Top]
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
