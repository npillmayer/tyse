/*
Package harfbuzz uses HarfBuzz converts text to sequencees of glyphs.

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package harfbuzz

import (
	"bytes"
	"encoding/binary"
	"io"
	"unicode"

	hbtt "github.com/benoitkugler/textlayout/fonts/truetype"
	hb "github.com/benoitkugler/textlayout/harfbuzz"
	hblang "github.com/benoitkugler/textlayout/language"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"github.com/npillmayer/tyse/engine/glyphing"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/language"
)

// tracer traces with key 'tyse.glyphs'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.glyphs")
}

// --- Type conversion -------------------------------------------------------

// Lang4HB returns a language tag as a HarfBuzz language.
func Lang4HB(l language.Tag) hblang.Language {
	return hblang.NewLanguage(l.String())
}

// Lang4HB returns a script as a HarfBuzz script.
func Script4HB(s language.Script) hblang.Script {
	b := []byte(s.String())
	b[0] = byte(unicode.ToLower(rune(b[0])))
	h := binary.BigEndian.Uint32(b)
	return hblang.Script(h)
}

// Direction4HB translates a direction to a HarfBuzz direction.
func Direction4HB(d glyphing.Direction) hb.Direction {
	switch d {
	case glyphing.LeftToRight:
		return hb.LeftToRight
	case glyphing.RightToLeft:
		return hb.RightToLeft
	case glyphing.TopToBottom:
		return hb.TopToBottom
	case glyphing.BottomToTop:
		return hb.BottomToTop
	}
	return hb.LeftToRight
}

// Feature4HB makes a typecast from an OpenType feature tag to a HarfBuzz truetype tag.
func Feature4HB(t ot.Tag) hbtt.Tag {
	return hbtt.Tag(t)
}

// FeatureRange4HB converts a feature range struct to a HarbBuzz Feature switch.
func FeatureRange4HB(frng glyphing.FeatureRange) hb.Feature {
	f := hb.Feature{
		Tag:   Feature4HB(frng.Feature),
		Start: frng.Start,
		End:   frng.End,
	}
	if frng.On {
		if frng.Arg > 0 {
			f.Value = uint32(frng.Arg)
		} else {
			f.Value = 1
		}
	}
	return f
}

// --- Shape -----------------------------------------------------------------

// Shape calls the HarfBuzz shaper.
//
// Shape shapes a sequence of code-points (runes), turning its Unicode characters to
// positioned glyphs. It will select a shape plan based on params, including the
// selected font, and the properties of the input text.
//
// If `params.Features` is not empty, it will be used to control the
// features applied during shaping. If two features have the same tag but
// overlapping ranges the value of the feature with the higher index takes
// precedence.
//
// params.Font must be set, otherwise no output is created.
//
// Clients may provide `buf` to avoid allocating memory by Shape. Shape will wrap it
// into the GlyphSequence returned.
//
var globalFontForTest *hb.Font

func Shape(text io.RuneReader, buf []glyphing.ShapedGlyph, context [][]rune, params glyphing.Params) (glyphing.GlyphSequence, error) {
	if text == nil || params.Font == nil {
		return glyphing.GlyphSequence{}, nil
	}
	// Prepare font
	var hb_font *hb.Font
	if globalFontForTest == nil {
		f := bytes.NewReader(params.Font.ScalableFontParent().Binary)
		hb_face, err := hbtt.Parse(f, true)
		if err != nil {
			return glyphing.GlyphSequence{}, err
		}
		hb_font = hb.NewFont(hb_face)
		globalFontForTest = hb_font
	} else {
		hb_font = globalFontForTest
	}
	hb_font.Ptem = params.Font.PtSize()
	// Prepare shaping parameters
	var hb_seqProps hb.SegmentProperties
	convertParams(&hb_seqProps, params)
	var features []hb.Feature = make([]hb.Feature, len(params.Features))
	for _, feat := range params.Features {
		features = append(features, FeatureRange4HB(feat))
	}
	// Prepare HarfBuzz buffer
	hb_buf := hb.NewBuffer()
	hb_buf.Props = hb_seqProps
	bytesBuf, offset, length := bufferText(text, context)
	runes := bytes.Runes(bytesBuf.Bytes())
	//tracer().Debugf("going to HarfBuzz-shape %v", runes)
	hb_buf.AddRunes(runes, offset, length)
	hb_buf.Shape(hb_font, features)
	// Prepare shaped output
	if buf == nil || len(buf) < len(hb_buf.Info) {
		buf = make([]glyphing.ShapedGlyph, len(hb_buf.Info))
	}
	seq := glyphing.GlyphSequence{
		Glyphs: buf,
	}
	// move HarfBuzz output to glyph sequence output
	sfont := params.Font.ScalableFontParent().SFNT
	var sfntBuf sfnt.Buffer
	for i, ginfo := range hb_buf.Info {
		gpos := &hb_buf.Pos[i]
		tracer().Debugf("[%3d] %q", i, ginfo.String())
		g := &buf[i]
		g.ClusterID = ginfo.Cluster
		g.GID = ot.GlyphIndex(ginfo.Glyph)
		g.XAdvance = dimen.DU(gpos.XAdvance) // TODO convert / caluculate
		g.YAdvance = dimen.DU(gpos.YAdvance)
		g.XOffset = dimen.DU(gpos.XOffset)
		g.YOffset = dimen.DU(gpos.YOffset)
		g.CodePoint = runes[g.ClusterID]
		bounds, adv, err := sfont.GlyphBounds(&sfntBuf, sfnt.GlyphIndex(g.GID), fixed.Int26_6(sfont.UnitsPerEm()), font.HintingNone)
		if err != nil {
			g.RawMetrics.Advance = sfnt.Units(adv)
			g.RawMetrics.BBox.MinX = sfnt.Units(bounds.Min.X)
			g.RawMetrics.BBox.MinY = sfnt.Units(bounds.Min.Y)
			g.RawMetrics.BBox.MaxX = sfnt.Units(bounds.Max.X)
			g.RawMetrics.BBox.MaxY = sfnt.Units(bounds.Max.Y)
			g.RawMetrics.LSB = g.RawMetrics.BBox.MinX
			g.RawMetrics.RSB = g.RawMetrics.Advance - g.RawMetrics.BBox.MaxX
		}
	}
	return seq, nil
}

// convertParams is a helper function to convert glyphing parameters to
// HarfBuzz's format.
func convertParams(hb_seqProps *hb.SegmentProperties, params glyphing.Params) {
	if params.Language != language.Und {
		hb_seqProps.Language = Lang4HB(params.Language)
	}
	var none language.Script
	if params.Script != none {
		hb_seqProps.Script = Script4HB(params.Script)
	}
	hb_seqProps.Direction = Direction4HB(params.Direction)
}

// bufferText buffers the input text of a call to Shape(…) as a bytes.Buffer.
// To conform to HarfBuzz's API, context is pre-/appended to the input runes.
//
// bufferText returns the start position of the input within the returned buffer,
// together with the input's length (= rune count).
func bufferText(text io.RuneReader, context [][]rune) (buf bytes.Buffer, off int, length int) {
	var bytesBuf bytes.Buffer
	var r rune
	if len(context) > 0 && len(context[0]) > 0 {
		for off, r = range context[0] {
			bytesBuf.WriteRune(r)
		}
	}
	var sz int
	var err error
	for {
		if r, sz, err = text.ReadRune(); sz == 0 || err != nil {
			break
		}
		length++
		bytesBuf.WriteRune(r)
	}
	if len(context) > 1 && len(context[1]) > 0 {
		for _, r = range context[1] {
			bytesBuf.WriteRune(r)
		}
	}
	return bytesBuf, off, length
}

/*
// Params collects shaping parameters.
type Params struct {
	Direction Direction       // writing direction
	Script    language.Script // 4-letter ISO 15924
	Language  language.Tag    // BCP 47 tag
	Font      *font.TypeCase  // font at a given point-size
}

// GlyphSequence contains a sequence of shaped glyphs.
type GlyphSequence struct {
	Glyphs  []ShapedGlyph // resulting sequence of glyphs
	W, H, D dimen.DU      // width, height, depth of bounding box
}
*/
