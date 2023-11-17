/*
Package opentype handles OpenType fonts.

# License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © Norbert Pillmayer <norbert@pillmayer.com>
*/
package opentype

import (
	"golang.org/x/image/font/sfnt"
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
