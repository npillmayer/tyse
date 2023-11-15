package otshape

import (
	"slices"
	"unicode/utf8"

	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"github.com/npillmayer/tyse/core/font/opentype/otquery"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/text/unicode/norm"
)

type ShapedGlyph struct {
	ginx    ot.GlyphIndex
	advance sfnt.Units
}

type Buffer []ShapedGlyph

func NewBuffer(n int) Buffer {
	if n < 0 {
		n = 0
	}
	return make([]ShapedGlyph, n)
}

// Get the glyph indices of the glyphs in a buffer.
// Allocates a slice of glyh-indices.
func (b Buffer) Glyphs() []ot.GlyphIndex {
	glyphs := make([]ot.GlyphIndex, len(b))
	for i, g := range b {
		glyphs[i] = g.ginx
	}
	return glyphs
}

// decide wether to compose or de-compose by default
func normalizersFor(script ot.Tag, lang ot.Tag) (norm.Form, norm.Form) {
	if prefersDecomposed(script, lang) {
		return norm.NFD, norm.NFC
	}
	return norm.NFC, norm.NFD
}

// representation reflects the best representation of a character for a font. A character
// consists of one or more Unicode code-points. The representation is NFC, NFD, or something
// in between.
type representation struct {
	nucleus  rune
	nucGlyph ot.GlyphIndex
	marks    []ot.GlyphIndex
}

// representation of unrepresentable character, .notdef
var norep = representation{nucleus: utf8.RuneError, nucGlyph: NOTDEF, marks: nil}

// Warning:
// ========
// All the following functions are sub-optimal in terms of garbage due to naive slice
// operations. As soon as they are stable, we will optimize them.

func (rep representation) mergeNucleus(ch rune, otf *ot.Font) representation {
	if rep.nucleus != 0 { // already a nucleus set -> merge
		var w int                                 // byte width of rune to produce
		runes := []rune{rep.nucleus, ch}          // create nuc x ch
		testNuc := norm.NFC.String(string(runes)) // NFC(nuc x ch) = ?
		ch, w = utf8.DecodeRuneInString(testNuc)  // check for first codepoint of ?
		if w >= len(testNuc) {                    // no luck, still 2 codepoints
			return norep
		}
	}
	glyph := otquery.GlyphIndex(otf, ch) // has NFC character a glyph?
	if glyph == NOTDEF {                 // no luck
		return norep
	}
	return representation{
		nucleus:  ch,
		nucGlyph: glyph,
		marks:    rep.marks,
	}
}

func (rep representation) appendMark(ch rune, otf *ot.Font) representation {
	glyph := otquery.GlyphIndex(otf, ch)
	if glyph == NOTDEF {
		return norep
	}
	return representation{
		nucleus:  rep.nucleus,
		nucGlyph: rep.nucGlyph,
		marks:    append(rep.marks, glyph),
	}
}

// codepoints is NFD
func (rep representation) representNFD(codepoints []byte, otf *ot.Font) representation {
	if len(codepoints) == 0 || rep.nucleus == utf8.RuneError {
		return rep
	}
	ch, w := utf8.DecodeRune(codepoints)
	if w == 0 || ch == utf8.RuneError { // this should never happen
		return norep
	}
	merged := rep.mergeNucleus(ch, otf).representNFD(codepoints[w:], otf)
	appended := rep.appendMark(ch, otf).representNFD(codepoints[w:], otf)
	// TODO: let client decide wether shorter or longer form is preferable
	if merged.nucGlyph != NOTDEF && len(merged.marks) <= len(appended.marks) {
		return merged
	}
	return appended
}

// codepoints is either fully NFC or NFD
func findRepresentation(codepoints []byte, otf *ot.Font, buf []ot.GlyphIndex) []ot.GlyphIndex {
	if buf == nil {
		buf = make([]ot.GlyphIndex, 0, 16)
	} else {
		buf = buf[:0]
	}
	if norm.NFC.IsNormal(codepoints) {
		ch, _ := utf8.DecodeRune(codepoints)
		glyph := otquery.GlyphIndex(otf, ch)
		if glyph != NOTDEF {
			buf = append(buf, glyph)
			return buf
		}
		codepoints = norm.NFD.Bytes(codepoints)
	}
	rep := representation{}.representNFD(codepoints, otf)
	buf = slices.Grow(buf, len(rep.marks)+1)
	buf = append(buf, rep.nucGlyph)
	buf = append(buf, rep.marks...)
	return buf
}

// Convert a buffer of codepoints to its initial glyph mapping.
// Does OpenType normalization, as explained here:
// https://github.com/n8willis/opentype-shaping-documents/blob/master/opentype-shaping-normalization.md
func (b Buffer) mapGlyphs(input string, otf *ot.Font, script ot.Tag, lang ot.Tag) int {
	buf := make([]ot.GlyphIndex, 0, 16)
	normalizerDefault, _ := normalizersFor(script, lang)
	tracer().Debugf("mapping glyphs for input string = %v", []byte(input))
	clear(b)
	var iterInput norm.Iter
	iterInput.InitString(normalizerDefault, input)
	var i, n int
	for !iterInput.Done() {
		codepoints := iterInput.Next() // get a sequence of code-points
		tracer().Debugf("read codepoints '%s' (%v)", string(codepoints), codepoints)
		glyphs := findRepresentation(codepoints, otf, buf)
		var glyph ot.GlyphIndex
		for i, glyph = range glyphs {
			b[n+i].ginx = glyph
			metrics := otquery.GlyphMetrics(otf, glyph)
			b[n+i].advance = metrics.Advance
		}
		n += i
	}
	return n
}
