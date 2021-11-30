package otquery

import (
	"errors"
	"strings"

	"github.com/npillmayer/tyse/core/font/opentype"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/text/language"
)

// TODO
func SupportsScript(otf *ot.Font, scr language.Script) (string, string) {
	t := otf.Table(ot.T("GSUB"))
	if t == nil {
		// do nothing
		return "", ""
	}
	gsub := t.Self().AsGSub()
	scrTag := ot.T(strings.ToLower(scr.String()))
	rec := gsub.ScriptList.LookupTag(scrTag)
	if rec.IsNull() {
		tracer().Debugf("cannot find script %s in font", scr.String())
	} else {
		tracer().Debugf("script %s is contained in GSUB", scr.String())
		s := rec.Navigate()
		for _, tag := range s.Map().AsTagRecordMap().Tags() {
			tracer().Debugf("tag = %s", tag.String())
			l := s.Map().AsTagRecordMap().LookupTag(tag)
			lsys := l.Navigate()
			tracer().Debugf("list = %v", lsys.List())
		}
	}
	return "DFLT", "DFLT"
}

// GlyphIndex returns the glyph index for a give code-point.
// If the code-point cannot be found, 0 is returned.
func GlyphIndex(otf *ot.Font, codepoint rune) ot.GlyphIndex {
	if c := otf.Table(ot.T("cmap")); c == nil {
		return 0
	} else {
		return c.Self().AsCMap().GlyphIndexMap.Lookup(codepoint)
	}
}

// CodePointForGlyph returns the code-point for a given glyph index.
// This is an inefficient operation: All code-points contained in the font
// are checked sequentially if they produce th given glyph.
func CodePointForGlyph(otf *ot.Font, gid ot.GlyphIndex) rune {
	if gid == 0 {
		return 0
	} else if c := otf.Table(ot.T("cmap")); c == nil {
		return 0
	} else {
		return c.Self().AsCMap().GlyphIndexMap.ReverseLookup(gid)
	}
}

// FontMetrics retrieves selected metrics of a font.
func FontMetrics(otf *ot.Font) (opentype.FontMetricsInfo, error) {
	metrics := opentype.FontMetricsInfo{}
	if hhea := otf.Table(ot.T("hhea")); hhea != nil {
		tracer().Debugf("hhea")
		b := hhea.Binary()
		metrics.Ascent = sfnt.Units(i16(b[4:]))
		metrics.Descent = sfnt.Units(i16(b[6:]))
		metrics.LineGap = sfnt.Units(i16(b[8:]))
		metrics.MaxAdvance = sfnt.Units(u16(b[8:]))
	}
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
	h := otf.Table(ot.T("head"))
	if h != nil {
		head := h.Self().AsHead()
		metrics.UnitsPerEm = sfnt.Units(head.UnitsPerEm)
	}
	if metrics.Ascent == 0 && metrics.Descent == 0 {
		return metrics, errors.New("cannot find metric information in font")
	}
	return metrics, nil
}

// GlyphMetrics retrieves metrics for a given glyph.
func GlyphMetrics(otf *ot.Font, gid ot.GlyphIndex) (opentype.GlyphMetricsInfo, error) {
	metrics := opentype.GlyphMetricsInfo{}
	//
	// table HMtx: advance width and left side bearing
	var hmtx *ot.HMtxTable
	if t := otf.Table(ot.T("hmtx")); t != nil {
		hmtx = t.Self().AsHMtx()
	}
	var maxp *ot.MaxPTable
	if t := otf.Table(ot.T("maxp")); t != nil {
		maxp = t.Self().AsMaxP()
	}
	if maxp == nil || hmtx == nil {
		return metrics, errors.New("no glyph metrics available")
	}
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
	//
	// RSB calculation: rsb = aw - (lsb + xMax - xMin)
	// From the spec:
	// If a glyph has no contours, xMax/xMin are not defined. The left side bearing indicated
	// in the 'hmtx' table for such glyphs should be zero.
	if !metrics.BBox.Empty() { // leave RSB for empty bboxes
		metrics.RSB = metrics.Advance - (metrics.LSB + metrics.BBox.Dx())
	}
	return metrics, nil
}

func u16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])<<0
}

func i16(b []byte) int16 {
	return int16(b[0])<<8 | int16(b[1])<<0
}

// func i32(b []byte) int32 {
// 	return int32(b[0])<<24 | int32(b[1])<<16 | int32(b[2])<<8 | int32(b[3])<<0
// }
