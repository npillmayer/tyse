package monospace

import (
	"fmt"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/engine/text"
	"github.com/npillmayer/uax/grapheme"
	"github.com/npillmayer/uax/uax11"
)

type msshape struct {
	em  dimen.Dimen
	dir text.Direction
	// graphemeSplitter *segment.Segmenter
	context *uax11.Context
}

// Shaper creates a shaper for monospace typesetting.
// An em-dimension may be given which will then be used for shaping text.
// If is is zero, a monospace must be provided with Shape(…).
func Shaper(em dimen.Dimen, context *uax11.Context) text.Shaper {
	sh := &msshape{
		em:  em,
		dir: text.LeftToRight,
	}
	if context == nil {
		sh.context = uax11.LatinContext
	}
	// onGraphemes := grapheme.NewBreaker()
	// sh.graphemeSplitter = segment.NewSegmenter(onGraphemes)
	grapheme.SetupGraphemeClasses()
	return sh
}

// Shape creates a glyph sequence from a text.
func (ms msshape) Shape(text string, typecase *font.TypeCase) text.GlyphSequence {
	if ms.em == 0 && typecase == nil {
		T().Errorf("monospace shaper has em=0 and not font provided => no output")
		return msglyphseq{}
	}
	if ms.em == 0 {
		panic("TODO shaping with monospace font not yet implemented")
	}
	//
	gstr := grapheme.StringFromString(text)
	if gstr.Len() == 0 {
		return msglyphseq{}
	}
	seq := msglyphseq{}
	l := gstr.Len()
	for i := 0; i < l; i++ {
		grphm := []byte(gstr.Nth(i))
		w := uax11.Width(grphm, ms.context)
		g := msglyph{
			grapheme: grphm,
			pos:      i,
			w:        dimen.Dimen(w) * ms.em,
		}
		seq.glyphs = append(seq.glyphs, g)
		seq.w += g.w
	}
	seq.h = 3 / 5 * ms.em
	seq.d = 2 / 5 * ms.em
	return seq
}

// SetScript does not do anything for monospace shapers.
func (ms msshape) SetScript(scr text.ScriptID) {
	//
}

// SetDirection sets the text direction.
func (ms *msshape) SetDirection(dir text.Direction) {
	ms.dir = dir
}

// SetLanguage does not do anything for monospace shapers.
func (ms msshape) SetLanguage(string) {}

// --- Glyphs ----------------------------------------------------------------

type msglyphseq struct {
	glyphs  []msglyph
	w, h, d dimen.Dimen
}

func (gseq msglyphseq) GlyphCount() int {
	return len(gseq.glyphs)
}

func (gseq msglyphseq) GetGlyphInfoAt(pos int) text.GlyphInfo {
	return gseq.glyphs[pos]
}

func (gseq msglyphseq) BBoxDimens() (dimen.Dimen, dimen.Dimen, dimen.Dimen) {
	return gseq.w, gseq.h, gseq.d
}

func (gseq msglyphseq) Font() *font.TypeCase {
	return nil
}

func (gseq msglyphseq) String() string {
	s := ""
	for _, g := range gseq.glyphs {
		s += g.String()
	}
	return s
}

var _ text.GlyphSequence = msglyphseq{}

type msglyph struct {
	glyph    rune
	grapheme []byte
	pos      int
	w        dimen.Dimen
}

func (g msglyph) Glyph() rune {
	return g.glyph
}

func (g msglyph) Cluster() int {
	return g.pos
}

func (g msglyph) XAdvance() dimen.Dimen {
	return g.w
}

func (g msglyph) YAdvance() dimen.Dimen {
	return 0
}

func (g msglyph) XPosition() dimen.Dimen {
	return 0
}

func (g msglyph) YPosition() dimen.Dimen {
	return 0
}

func (g msglyph) String() string {
	return fmt.Sprintf("['%#U' %d]", g.glyph, g.w)
}

var _ text.GlyphInfo = msglyph{}

// ---------------------------------------------------------------------------

// type rr struct {
// 	runes []rune
// 	pos   int
// }

// func (reader *rr) ReadRune() (r rune, size int, err error) {
// 	if reader.pos == len(reader.runes) {
// 		return utf8.RuneError, 0, io.EOF
// 	}
// 	r = reader.runes[reader.pos]
// 	size = utf8.RuneLen(r)
// 	reader.pos++
// 	return
// }
