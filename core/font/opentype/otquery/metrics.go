package otquery

import (
	"github.com/npillmayer/tyse/core/font/opentype"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"golang.org/x/image/font/sfnt"
)

// --- Font Information -------------------------------------------------

// FontSupportsScript returns a tuple (script-tag, language-tag) for a given input
// of a script tag and a language tag. If the language has no special support in the
// font, DFLT will be returned. If the script has no support in the font,
// DFLT will be returned for the script.
func FontSupportsScript(otf *ot.Font, scr ot.Tag, lang ot.Tag) (ot.Tag, ot.Tag) {
	gsub := otf.Layout.GSub
	rec := gsub.ScriptList.Map().LookupTag(scr)
	if rec.IsNull() {
		tracer().Infof("cannot find script %s in font", scr.String())
		return ot.DFLT, ot.DFLT
	}
	tracer().Debugf("script %s is contained in GSUB", scr.String())
	s := rec.Navigate()
	for _, tag := range s.Map().AsTagRecordMap().Tags() {
		if tag.String() == lang.String() {
			return scr, lang
		}
	}
	return scr, ot.DFLT
}

// FontMetrics retrieves selected metrics of a font.
func FontMetrics(otf *ot.Font) opentype.FontMetricsInfo {
	metrics := opentype.FontMetricsInfo{}
	hhea := otf.Table(ot.T("hhea"))
	b := hhea.Binary()
	metrics.Ascent = sfnt.Units(i16(b[4:]))
	metrics.Descent = sfnt.Units(i16(b[6:]))
	metrics.LineGap = sfnt.Units(i16(b[8:]))
	metrics.MaxAdvance = sfnt.Units(u16(b[8:]))
	if metrics.Ascent == 0 && metrics.Descent == 0 {
		if os2 := otf.Table(ot.T("OS/2")); os2 != nil {
			tracer().Debugf("OS/2")
			b := os2.Binary()
			a := sfnt.Units(i16(b[68:]))
			if a > metrics.Ascent {
				tracer().Debugf("override of ascent: %d -> %d", metrics.Ascent, a)
				metrics.Ascent = sfnt.Units(a)
			}
			d := sfnt.Units(i16(b[70:]))
			if d < metrics.Descent {
				tracer().Debugf("override of descent: %d -> %d", metrics.Descent, d)
				metrics.Descent = sfnt.Units(d)
			}
		}
	}
	head := otf.Table(ot.T("head")).Self().AsHead() // Head is a required table
	metrics.UnitsPerEm = sfnt.Units(head.UnitsPerEm)
	return metrics
}

// --- Glyph Routines --------------------------------------------------------

// GlyphIndex returns the glyph index for a give code-point.
// If the code-point cannot be found, 0 is returned.
//
// From the OpenType specification: character codes that do not correspond to any glyph in
// the font should be mapped to glyph index 0. The glyph at this location must be a special
// glyph representing a missing character, commonly known as '.notdef'.
func GlyphIndex(otf *ot.Font, codepoint rune) ot.GlyphIndex {
	return otf.CMap.GlyphIndexMap.Lookup(codepoint)
}

// CodePointForGlyph returns the code-point for a given glyph index.
//
// This is an inefficient operation: All code-points contained in the font's CMap
// are checked sequentially if they produce the given glyph.
// If the glyph index does not correspond to a code-point, 0 is returned.
func CodePointForGlyph(otf *ot.Font, gid ot.GlyphIndex) rune {
	if gid == 0 {
		return 0
	}
	return otf.CMap.GlyphIndexMap.ReverseLookup(gid)
}

// GlyphMetrics retrieves metrics for a given glyph.
func GlyphMetrics(otf *ot.Font, gid ot.GlyphIndex) opentype.GlyphMetricsInfo {
	metrics := opentype.GlyphMetricsInfo{}
	//
	// table HMtx: advance width and left side bearing
	hmtx := otf.Table(ot.T("hmtx")).Self().AsHMtx() // required table in OpenType
	// table MaxP: number of glyphs
	maxp := otf.Table(ot.T("maxp")).Self().AsMaxP() // required table in OpenType
	mtxcnt := hmtx.NumberOfHMetrics
	diff := maxp.NumGlyphs - mtxcnt
	//tracer().Debugf("#glyphs=%d, #hmtx=%d, diff=%d", maxp.NumGlyphs, mtxcnt, diff)
	if gid < ot.GlyphIndex(mtxcnt) {
		l := ot.ParseList(hmtx.Binary(), mtxcnt, 4)
		entry := l.Get(int(gid))
		metrics.Advance = sfnt.Units(u16(entry.Bytes()))
		metrics.LSB = sfnt.Units(i16(entry.Bytes()[2:]))
	} else { // advance repetition of last advance in hmtx
		l := ot.ParseList(hmtx.Binary(), mtxcnt, 4)
		lastEntry := l.Get(mtxcnt - 1)
		metrics.Advance = sfnt.Units(u16(lastEntry.Bytes()))
		l = ot.ParseList(hmtx.Binary()[mtxcnt*4:], diff, 2)
		entry := l.Get(int(gid) - mtxcnt)
		metrics.LSB = sfnt.Units(i16(entry.Bytes()))
	}
	//
	// table glyf: bounding box
	if glyf := otf.Table(ot.T("glyf")); glyf != nil {
		if lo := otf.Table(ot.T("loca")); lo != nil {
			loca := lo.Self().AsLoca()
			loc := loca.IndexToLocation(gid)
			b := glyf.Binary()[loc:]
			metrics.BBox = opentype.BoundingBox{
				MinX: sfnt.Units(i16(b[2:])),
				MinY: sfnt.Units(i16(b[4:])),
				MaxX: sfnt.Units(i16(b[6:])),
				MaxY: sfnt.Units(i16(b[8:])),
			}
		}
	}
	// RSB calculation: rsb = aw - (lsb + xMax - xMin)
	// From the spec:
	// If a glyph has no contours, xMax/xMin are not defined. The left side bearing indicated
	// in the 'hmtx' table for such glyphs should be zero.
	if !metrics.BBox.Empty() { // leave RSB for empty bboxes
		metrics.RSB = metrics.Advance - (metrics.LSB + metrics.BBox.Dx())
	}
	return metrics
}

// --- Helpers ----------------------------------------------------------

func u16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])<<0
}

func i16(b []byte) int16 {
	return int16(b[0])<<8 | int16(b[1])<<0
}

// func i32(b []byte) int32 {
// 	return int32(b[0])<<24 | int32(b[1])<<16 | int32(b[2])<<8 | int32(b[3])<<0
// }
