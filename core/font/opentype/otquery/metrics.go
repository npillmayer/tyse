package otquery

import (
	"errors"
	"image"
	"strings"

	"github.com/npillmayer/tyse/core/font/opentype/ot"
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

// FontMetricsInfo contains selected metric information for a font.
type FontMetricsInfo struct {
	UnitsPerEm      int    // ad-hoc units per em
	Ascent, Descent int16  // ascender and descender
	MaxAdvance      uint16 // maximum advance width value in 'hmtx' table
	LineGap         int16  // typographic line gap
}

// FontMetrics retrieves selected metrics of a font.
func FontMetrics(otf *ot.Font) (FontMetricsInfo, error) {
	metrics := FontMetricsInfo{}
	if hhea := otf.Table(ot.T("hhea")); hhea != nil {
		tracer().Debugf("hhea")
		b := hhea.Binary()
		metrics.Ascent = i16(b[4:])
		metrics.Descent = i16(b[6:])
		metrics.LineGap = i16(b[8:])
		metrics.MaxAdvance = u16(b[8:])
	}
	if metrics.Ascent == 0 && metrics.Descent == 0 {
		if os2 := otf.Table(ot.T("OS/2")); os2 != nil {
			tracer().Debugf("OS/2")
			b := os2.Binary()
			a := i16(b[68:])
			if a > metrics.Ascent {
				tracer().Debugf("override of ascent: %d -> %d", metrics.Ascent, a)
				metrics.Ascent = a
			}
			d := i16(b[70:])
			if d < metrics.Descent {
				tracer().Debugf("override of descent: %d -> %d", metrics.Descent, d)
				metrics.Descent = d
			}
		}
	}
	h := otf.Table(ot.T("head"))
	if h != nil {
		head := h.Self().AsHead()
		metrics.UnitsPerEm = int(head.UnitsPerEm)
	}
	if metrics.Ascent == 0 && metrics.Descent == 0 {
		return metrics, errors.New("cannot find metric information in font")
	}
	return metrics, nil
}

// GlyphMetricsInfo contains all the metric information for a glyph.
type GlyphMetricsInfo struct {
	Advance  uint16          // advance width
	LSB, RSB int16           // side bearings
	BBox     image.Rectangle // bounding box
}

// GlyphMetrics retrieves metrics for a given glyph.
func GlyphMetrics(otf *ot.Font, gid ot.GlyphIndex) (GlyphMetricsInfo, error) {
	metrics := GlyphMetricsInfo{}
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
		metrics.Advance = u16(entry.Bytes())
		metrics.LSB = i16(entry.Bytes()[2:])
	} else { // advance repetition of last advance in hmtx
		l := ot.ParseList(hmtx.Binary(), mtxcnt, 4)
		lastEntry := l.Get(mtxcnt - 1)
		metrics.Advance = u16(lastEntry.Bytes())
		l = ot.ParseList(hmtx.Binary()[mtxcnt*4:], diff, 2)
		entry := l.Get(int(gid) - mtxcnt)
		metrics.LSB = i16(entry.Bytes())
	}
	//
	// table glyf: bounding box
	if glyf := otf.Table(ot.T("glyf")); glyf != nil {
		if lo := otf.Table(ot.T("loca")); lo != nil {
			loca := lo.Self().AsLoca()
			loc := loca.IndexToLocation(gid)
			b := glyf.Binary()[loc:]
			metrics.BBox = image.Rect(
				int(i16(b[2:])),
				int(i16(b[4:])),
				int(i16(b[6:])),
				int(i16(b[8:])),
			)
		}
	}
	//
	// RSB calculation: rsb = aw - (lsb + xMax - xMin)
	// From the spec:
	// If a glyph has no contours, xMax/xMin are not defined. The left side bearing indicated
	// in the 'hmtx' table for such glyphs should be zero.
	if !metrics.BBox.Empty() { // leave RSB for empty bboxes
		metrics.RSB = int16(metrics.Advance) - (metrics.LSB + int16(metrics.BBox.Dx()))
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
