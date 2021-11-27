/*
Package opentype handles OpenType fonts.

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package opentype

import (
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// --- Font and glyph metrics ------------------------------------------------

// FontMetricsInfo contains selected metric information for a font.
type FontMetricsInfo struct {
	UnitsPerEm      sfnt.Units // ad-hoc units per em
	Ascent, Descent sfnt.Units // ascender and descender
	MaxAdvance      sfnt.Units // maximum advance width value in 'hmtx' table
	LineGap         sfnt.Units // typographic line gap
}

// GlyphMetricsInfo contains all the metric information for a glyph.
type GlyphMetricsInfo struct {
	Advance  sfnt.Units  // advance width
	LSB, RSB sfnt.Units  // side bearings
	BBox     BoundingBox // bounding box
}

// BoundingBox describes the bounding box of a glyph.
type BoundingBox struct {
	MinX, MinY sfnt.Units
	MaxX, MaxY sfnt.Units
}

// Empty is a predicate: has this box a zero area?
func (bbox BoundingBox) Empty() bool {
	return bbox.MaxX-bbox.MinX == 0 || bbox.MaxY-bbox.MinY == 0
}

// Dx is the horizontal extent of this box.
func (bbox BoundingBox) Dx() sfnt.Units {
	return bbox.MaxX - bbox.MinX
}

// Dy is the vertical extent of this box.
func (bbox BoundingBox) Dy() sfnt.Units {
	return bbox.MaxY - bbox.MinY
}

// ---------------------------------------------------------------------------

/*
u/em   = 2000
_em    = 12 pt  = 0,1666 in
_dpi   = 120
=>
_d/_em = 120 * 0,1666 = 19,992 pixels per em
=>
u1     = 150
_u1    = 150 / _d/_em  = 7,503  pixels

Beispiel:
PT  = 12
DPI = 72
_d/_em = gtx.Px(DPI) * (PT / 72.27)
=> gtx.Px(12)  vereinfacht bei dpi = 72
*/

// PtIn is 72.27, i.e. printer's points per inch.
var PtIn fixed.Int26_6 = fixed.I(27)/100 + fixed.I(72)

// PpEm calculates a ppem value for a given font point-size and an output resolution (dpi).
func PpEm(ptSize fixed.Int26_6, dpi float32) fixed.Int26_6 {
	_dpi := fixed.Int26_6(dpi * 64)
	return _dpi * (ptSize / PtIn)
}

func RasterCoords(otf *ot.Font, ptSize fixed.Int26_6, u sfnt.Units, dpi float32) fixed.Int26_6 {
	ppem := PpEm(ptSize, dpi)
	_u := fixed.I(int(u))
	return _u / ppem
}
