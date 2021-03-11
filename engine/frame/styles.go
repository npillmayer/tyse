package frame

/*
BSD 3-Clause License

Copyright (c) 2020–21, Norbert Pillmayer
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
   contributors may be used to endorse or promote products derived from
   this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"image/color"

	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/uax/bidi"
)

// StyleSet is a type to hold CSS-styles/properties for runs of text (of a paragraph).
type StyleSet struct {
	Props      *style.PropertyMap
	EmbBidiDir bidi.Direction // embedding bidi text direction
}

// String is part of interface cords.styled.Style.
func (set StyleSet) String() string {
	return "<style>"
}

// Equals is part of interface cords.styled.Style, not intended for client usage.
func (set StyleSet) Equals(other styled.Style) bool {
	if o, ok := other.(StyleSet); ok {
		if o.Props == set.Props {
			return true
		}
	}
	return false
}

var _ styled.Style = StyleSet{}

// ---------------------------------------------------------------------------

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

/*
We take the following CSS properties as relevant for line boxes / span

margin-left
margin-right
padding-left
padding-right
white-space
border-left
border-right
border-left-width
border-right-width
border-left-color
border-right-color
border-left-style
border-right-style
font-family
font-style
font-variant
font-size
strong
b
i
em
small
marked
color
background-color
direction
tabsize
text-indentation
line-height  // for strut ?
text-decoration
word-break
white-space

For line box building

border-top
border-bottom
border-top-width
border-bottom-width
border-top-color
border-bottom-color
border-top-style
border-bottom-style
border-top-left-radius
border-top-right-radius
border-bottom-left-radius
border-bottom-right-radius
list-style-type
list-style-position
list-style-image
text-alignment
text-overflow
word-wrap

*/

func (set StyleSet) Styles() *style.PropertyMap {
	return set.Props
}

func (set StyleSet) Parindent() dimen.Dimen {
	return 0
}

func (set StyleSet) Space() (dimen.Dimen, dimen.Dimen, dimen.Dimen) {
	// respect pre WS formatting
	return 5 * dimen.BP, 8 * dimen.BP, 4 * dimen.BP
}

func (set StyleSet) Whitespace(ws string) string {
	return " "
}

func (set StyleSet) BidiDir() bidi.Direction {
	return bidi.LeftToRight
}

func (set StyleSet) Font() *font.TypeCase {
	return font.NullTypeCase()
}
