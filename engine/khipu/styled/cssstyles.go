package styled

import (
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/engine/dom/cssom/style"
	"github.com/npillmayer/tyse/engine/text"
)

/*
We take the following CSS properties as relevant for line boxes / span

For khipu-enconding

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

func (set Set) Styles() *style.PropertyMap {
	return set.styles
}

func (set Set) Parindent() dimen.Dimen {
	return 0
}

func (set Set) Space() (dimen.Dimen, dimen.Dimen, dimen.Dimen) {
	// respect pre WS formatting
	return 5 * dimen.BP, 8 * dimen.BP, 4 * dimen.BP
}

func (set Set) Whitespace(ws string) string {
	return " "
}

func (set Set) BidiDir() text.Direction {
	return text.LeftToRight
}

func (set Set) Font() *font.TypeCase {
	return font.NullTypeCase()
}
