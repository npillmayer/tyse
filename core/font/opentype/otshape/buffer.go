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

const ( // for different scripts or script/language combinations shapers may prefer NFC or NFD
	PREFER_COMPOSED   = 0
	PREFER_DECOMPOSED = 1
)

// Decide wether to compose or de-compose by default.
// Returns the primary normalizer and its opposite.
func normalizersFor(script ot.Tag, lang ot.Tag) (norm.Form, int) {
	if prefersDecomposed(script, lang) {
		return norm.NFD, PREFER_DECOMPOSED
	}
	return norm.NFC, PREFER_COMPOSED
}

/*
OpenType normalization is different from Unicode normalization. A good description of the
challenges may be found here:
// https://github.com/n8willis/opentype-shaping-documents/blob/master/opentype-shaping-normalization.md

Some statements from the document:
Normalization for OpenType shaping closely follows the Unicode normalization model, but it
takes place in the context of a known text run and a specific active font.
As a result, OpenType shaping takes the text context and available font contents into account,
making decisions intended to result in the best possible output to the shaping process.


1. Different shaping models can request different preferred formats (composed or decomposed)
   as output
2. Individual decomposition and recomposition mappings will not be applied if doing so would
   result in a codepoint for which the active font does not provide a glyph
3. Additional decompositions and recompositions not included in Unicode are supported,
   including the decomposition of multi-part dependent vowels (matras) in several Indic
   and Brahmic-derived scripts as well as arbitrary decompositions and compositions implemented
   in ccmp and locl GSUB lookups

Class `representation` will deal with item (2). A good starting place would be
https://www.w3.org/wiki/I18N/CanonicalNormalizationIssues. The article uses the following
example of a combining character sequence that has multiple canonical representations:

a) Ệ (U+1EC6) [NFC]
b) Ê ◌̣ (U+00CA-U+0323)
c) Ẹ ◌̂ (U+1EB8-U+0302)
d) E ◌̂ ◌̣ (U+0045-U+0302-U+0323)
e) E ◌̣ ◌̂ (U+0045-U+0323-U+0302) [NFD]

(Note: if the example looks like garbage for you, your font may not support all the necessary glyphs.)
Class `representation` will choose the correct representation in accordance with the active font.
*/

// representation reflects the best representation of a character for a font. A character
// consists of one or more Unicode code-points. The representation will be NFC, NFD, or something
// in between.
//
// In the example above, if the best representation would be option (c), the nucleus would
// be 'Ẹ' (U+1EB8) and marks would be [ ◌̂ ] (U+0302). `nucGlyph` would be the glyph in the
// active font for U+1EB8. The glyph would be guaranteed to be defined (not `.notdef`).
type representation struct {
	nucleus  rune
	nucGlyph ot.GlyphIndex
	marks    []ot.GlyphIndex
}

// representation of unrepresentable character, .notdef
var norep = representation{nucleus: utf8.RuneError, nucGlyph: NOTDEF, marks: nil}

// findRepresentation finds the representation of a character best suited with an active
// font. In the example above, let's assume that the script in use dictated that all characters
// are initially NFC (composed), i.e. 'Ệ' (U+1EC6) has been the pre-selected representation
// of the character. findRepresentation would then check if U+1EC6 is present in font `otf`.
// If it isn't, all five combinations would then be evaluated and -- if a suitable one can
// be found -- one of (b) to (e) returned, `norep` otherwise.
//
// a) Ệ (U+1EC6) [NFC]
// b) Ê ◌̣ (U+00CA-U+0323)
// c) Ẹ ◌̂ (U+1EB8-U+0302)           assume this would be suited
// d) E ◌̂ ◌̣ (U+0045-U+0302-U+0323)
// e) E ◌̣ ◌̂ (U+0045-U+0323-U+0302) [NFD]
//
// The caller sends `codepoints“ either fully NFC or fully NFD. A sequence of glyph indices
// will be returned; in our example: [ Glyph(U+1EB8) Glyph(U+0302) ].
func findRepresentation(codepoints []byte, otf *ot.Font, buf []ot.GlyphIndex, flag int) []ot.GlyphIndex {
	//tracer().Debugf("find representation of %v '%s'", codepoints, string(codepoints))
	if buf == nil {
		buf = make([]ot.GlyphIndex, 0, 16)
	} else {
		buf = buf[:0]
	}
	if flag == PREFER_COMPOSED && norm.NFC.IsNormal(codepoints) { // character in composed format, i.e. single codepoint
		ch, w := utf8.DecodeRune(codepoints)
		assert(w == len(codepoints), "expected codepoints to be single NFC rune")
		glyph := otquery.GlyphIndex(otf, ch)
		if glyph != NOTDEF { // glyph is present in font -> return it
			buf = append(buf, glyph)
			return buf
		}
	}
	codepoints = norm.NFD.Bytes(codepoints) // maximally de-compose the character
	rep := representation{}.representNFD(codepoints, otf, flag)
	buf = slices.Grow(buf, len(rep.marks)+1)
	buf = append(buf, rep.nucGlyph) // concatenate nuclear glyph and marks
	buf = append(buf, rep.marks...)
	return buf
}

// Warning:
// ========
// All the following functions are sub-optimal in terms of garbage due to naive slice
// operations. As soon as they are stable, we will optimize them.

// At this point, `codepoints` is fully NFD. We'll have to check for each code-point if there
// is a glyph in the font for it. Otherwise we need to find combined glyph which represents
// a valid combination and includes the code-point.
//
// In our example, we receive NFD form "E ◌̣ ◌̂ (U+0045-U+0323-U+0302) [NFD]". Now we verify
// that 'E' is present in font `otf`. If it isn't, we'll have to move on to test for a glyph
// for "E + ◌̣ = Ẹ (U+1EB8)", which means merging to a new nucleus. Testing for all variants
// will be done recursively, one arm of the recursion testing for merging nucleus and the
// other arm testing for the mark glyph and appending it to the mark list.
func (rep representation) representNFD(codepoints []byte, otf *ot.Font, flag int) representation {
	//tracer().Debugf("representNFD = %v, codepoints = %v", rep, codepoints)
	if len(codepoints) == 0 || rep.nucleus == utf8.RuneError {
		return rep
	}
	ch, w := utf8.DecodeRune(codepoints)
	//tracer().Debugf("w = %d, ch = %#U", w, ch)
	assert(w != 0 || ch != utf8.RuneError, "did not expect illegal code-point here")
	assert(norm.NFD.IsNormal(codepoints), "at this point only NFD is expected")
	//tracer().Debugf("-> calling merge")
	merged := rep.mergeNucleus(ch, otf).representNFD(codepoints[w:], otf, flag)
	//tracer().Debugf("- merged = %v", merged)
	//tracer().Debugf("-> calling append")
	appended := rep.appendMark(ch, otf).representNFD(codepoints[w:], otf, flag)
	//tracer().Debugf("- appended = %v", merged)
	if merged.nucGlyph != NOTDEF {
		if appended.nucGlyph == NOTDEF {
			return merged
		} else if flag == PREFER_COMPOSED && len(merged.marks) <= len(appended.marks) {
			return merged
		}
	}
	return appended
}

// mergeNucleus tests for the possibilty of merging the current nucleus with the next
// codepoint and -- if possible -- tests for presence of a glyph in the font.
//
// This would be case "E + ◌̣ = Ẹ (U+1EB8)" -> Glyph(U+1EB8)  for our example.
func (rep representation) mergeNucleus(ch rune, otf *ot.Font) representation {
	if rep.nucleus != 0 { // already a nucleus set -> merge
		runes := []rune{rep.nucleus, ch}          // create nuc x ch
		testNuc := norm.NFC.String(string(runes)) // NFC(nuc x ch) = ?
		ch, _ = utf8.DecodeRuneInString(testNuc)  // check for first codepoint of ? is merged
		if runes[0] == []rune(testNuc)[0] {       // no luck, still 2 codepoints
			//tracer().Debugf("  pre: %s|%v, post: %s|%v", string(runes), runes, testNuc, []rune(testNuc))
			//tracer().Debugf("  cannot merge)"
			return norep
		}
	}
	glyph := otquery.GlyphIndex(otf, ch) // has NFC character a glyph?
	if glyph == NOTDEF {                 // no luck
		//tracer().Debugf("  no glyph for %#U", ch)
		return norep
	}
	//tracer().Debugf("  merged rune = %#U", ch)
	return representation{
		nucleus:  ch,    // we have a new nucleus
		nucGlyph: glyph, // and a new glyph for it
		marks:    rep.marks,
	}
}

// appendMark tests for presence of the next code-point (which should be a mark) in
// the font and appends the corresponding glyph to the list of marks.
//
// This would be case "E + ◌̣" -> marks={ ◌̣ Glyph(U+1EB8) } for our example.
func (rep representation) appendMark(ch rune, otf *ot.Font) representation {
	if rep.nucGlyph == NOTDEF {
		//tracer().Debugf("  mark may not be the first code-point")
		return norep // mark may not be the first code-point
	}
	glyph := otquery.GlyphIndex(otf, ch)
	if glyph == NOTDEF {
		//tracer().Debugf("  mark glyph not defined")
		return norep
	}
	return representation{
		nucleus:  rep.nucleus,
		nucGlyph: rep.nucGlyph,
		marks:    append(rep.marks, glyph),
	}
}

// Convert a buffer of codepoints to its initial glyph mapping.
// During the mapping we perform OpenType normalization, as explained here:
// https://github.com/n8willis/opentype-shaping-documents/blob/master/opentype-shaping-normalization.md
//
// TODO: comment this extensively !
func (b Buffer) mapGlyphs(input string, otf *ot.Font, script ot.Tag, lang ot.Tag) int {
	buf := make([]ot.GlyphIndex, 0, 16)
	normalizerDefault, normFlag := normalizersFor(script, lang)
	//tracer().Debugf("mapping glyphs for input string = %v", []byte(input))
	clear(b)
	//tracer().Debugf("shaping buffer is of length %d", len(b))
	var iterInput norm.Iter
	iterInput.InitString(normalizerDefault, input)
	var n int
	for !iterInput.Done() {
		codepoints := iterInput.Next() // get a sequence of code-points
		tracer().Debugf("read codepoints '%s' (%v)", string(codepoints), codepoints)
		glyphs := findRepresentation(codepoints, otf, buf, normFlag)
		tracer().Debugf("glyphs = %v", glyphs)
		var glyph ot.GlyphIndex
		for _, glyph = range glyphs {
			b[n].ginx = glyph
			metrics := otquery.GlyphMetrics(otf, glyph)
			b[n].advance = metrics.Advance
			n++
		}
	}
	return n
}
