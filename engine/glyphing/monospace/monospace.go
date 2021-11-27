package monospace

import (
	"io"
	"unicode/utf8"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/glyphing"
	"github.com/npillmayer/uax/grapheme"
	"github.com/npillmayer/uax/segment"
	"github.com/npillmayer/uax/uax11"
	"golang.org/x/text/language"
)

type msshape struct {
	em               dimen.DU
	dir              glyphing.Direction
	graphemeSplitter *segment.Segmenter
	context          *uax11.Context
}

// Shaper creates a shaper for monospace typesetting.
// An em-dimension may be given which will then be used for shaping text.
// If is is zero, it will be set to 10pt.
func Shaper(em dimen.DU, context *uax11.Context) glyphing.Shaper {
	if em == 0 {
		em = 10 * dimen.PT
	}
	sh := &msshape{
		em:  em,
		dir: glyphing.LeftToRight,
	}
	if context == nil {
		sh.context = uax11.LatinContext
	}
	onGraphemes := grapheme.NewBreaker(1)
	sh.graphemeSplitter = segment.NewSegmenter(onGraphemes)
	grapheme.SetupGraphemeClasses()
	return sh
}

// Shape creates a glyph sequence from a text.
func (ms msshape) Shape(text io.RuneReader, buf []glyphing.ShapedGlyph, ctx [][]rune, p glyphing.Params) (glyphing.GlyphSequence, error) {
	if text == nil {
		return glyphing.GlyphSequence{}, nil
	}
	seq := glyphing.GlyphSequence{Glyphs: buf}
	if seq.Glyphs == nil {
		seq.Glyphs = make([]glyphing.ShapedGlyph, 0, 256)
	}
	ms.graphemeSplitter.Init(text)
	i := 0
	for ms.graphemeSplitter.Next() {
		grphm := ms.graphemeSplitter.Bytes()
		w := uax11.Width(grphm, ms.context)
		codepoint, _ := utf8.DecodeRune(grphm)
		g := glyphing.ShapedGlyph{
			XAdvance:  dimen.DU(w) * ms.em,
			ClusterID: i,
			CodePoint: codepoint,
		}
		seq.Glyphs = append(seq.Glyphs, g)
		seq.W += g.XAdvance
		i++
	}
	seq.H = 3 / 5 * ms.em
	seq.D = 2 / 5 * ms.em
	return seq, nil
}

// SetScript does not do anything for monospace shapers.
func (ms msshape) SetScript(scr language.Script) {
	//
}

// SetDirection sets the text direction.
func (ms *msshape) SetDirection(dir glyphing.Direction) {
	ms.dir = dir
}

// SetLanguage does not do anything for monospace shapers.
func (ms msshape) SetLanguage(language.Tag) {}
