package khipu

/*
BSD License

Copyright (c) 2017–20, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of the software nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"bufio"
	"io"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/locate"
	params "github.com/npillmayer/tyse/core/parameters"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/glyphing"
	"github.com/npillmayer/uax"
	"github.com/npillmayer/uax/bidi"
	"github.com/npillmayer/uax/segment"
	"github.com/npillmayer/uax/uax14"
	"github.com/npillmayer/uax/uax29"
	"golang.org/x/text/unicode/norm"
)

// A TypesettingPipeline consists of steps to produce a khipu from text.
type TypesettingPipeline struct {
	input       io.RuneReader
	linewrap    *uax14.LineWrap
	wordbreaker *uax29.WordBreaker
	segmenter   *segment.Segmenter
	words       *segment.Segmenter
}

type typEnv struct { // typesetting environment
	shaper   glyphing.Shaper
	pipeline *TypesettingPipeline
	regs     *params.TypesettingRegisters
	levels   *bidi.ResolvedLevels
}

type styledItem struct {
	offset             uint64
	end                uint64
	from, to           uint64
	prevStyles, styles styled.Style
}

func EncodeParagraph(para *styled.Paragraph, startpos uint64, shaper glyphing.Shaper,
	pipeline *TypesettingPipeline, regs *params.TypesettingRegisters) (*Khipu, error) {
	//
	if regs == nil {
		regs = params.NewTypesettingRegisters()
	}
	env := typEnv{
		shaper:   shaper,
		pipeline: pipeline,
		regs:     regs,
		levels:   para.BidiLevels(),
	}
	text := para.Raw().Reader()
	//paraLen := para.Raw().Len()
	env.pipeline = PrepareTypesettingPipeline(text, env.pipeline)
	var result *Khipu = NewKhipu()
	T().Debugf("------------ start of para -----------")
	//T().Debugf("para text = '%s'", para.Raw().String())
	para.EachStyleRun(func(content string, sty styled.Style, pos, length uint64) error {
		item := styledItem{
			offset: para.Offset,
			end:    para.Offset + para.Raw().Len(),
			from:   startpos + pos,
			to:     startpos + pos + length,
			styles: sty,
		}
		T().Debugf("--- encoding run '%s'", content)
		k, err := encodeRun(text, item, env)
		if err != nil {
			return err
		}
		result = result.AppendKhipu(k)
		return nil
	})
	T().Debugf("------------- end of para ------------")
	return result, nil
}

func encodeRun(text io.Reader, item styledItem, env typEnv) (k *Khipu, err error) {
	k = NewKhipu()                   // will be return value
	stopper := int64(item.to)        // we won't read beyond end of item run
	if uint64(stopper) >= item.end { // except when at end of paragraph
		stopper = math.MaxInt64 // set stopper to unreachable value
	}
	seg := env.pipeline.segmenter
	for seg.BoundedNext(stopper) {
		segment := seg.Text()
		p := penlty(seg.Penalties())
		T().Debugf("next segment = '%s'\twith penalties %d|%d", segment, p.p1, p.p2)
		item.from = item.to
		item.to += uint64(len(segment))
		kfrag, err := encodeSegment(segment, p, item, env)
		// if regs.N(params.P_MINHYPHENLENGTH) < dimen.Infty {
		// 	HyphenateTextBoxes(k, pipeline, regs)
		// }
		if err != nil {
			break
		}
		k = k.AppendKhipu(kfrag)
	}
	T().Debugf("------------- end of run -------------")
	return k, err
}

func encodeSegment(segm string, p penalties, item styledItem, env typEnv) (*Khipu, error) {
	//
	if p.breaksAtSpace() && isspace(segm) {
		return encodeSpace(segm, p, item.styles, env.regs), nil
	}
	if p.canWrapLine() && p.breaksAtSpace() {
		// line wrap at space
		// b := NewTextBox(seg.Text(), textpos)
		// khipu.AppendKnot(b).AppendKnot(Penalty(dimen.Infty))
		b := encodeText(segm, item, env)
		b.AppendKnot(Penalty(p.p1))
		return b, nil
	}
	if p.canWrapLine() { // line wrap without space
		// identified as a possible line break, but no space
		// insert explicit discretionary '\-' penalty
		k := NewKhipu().AppendKnot(Penalty(env.regs.N(params.P_HYPHENPENALTY)))
		return k, nil
	}
	// no line wrap and no break at space: inhibit break with infinite penalty
	// close a text box which is not a possible line wrap position
	// b := NewTextBox(seg.Text(), textpos)
	// pen := Penalty(dimen.Infty)
	// khipu.AppendKnot(b).AppendKnot(pen)
	T().Debugf("no line wrap possible, encode unbreakable text")
	b := encodeText(segm, item, env)
	b.AppendKnot(Penalty(dimen.Infinity))
	return b, nil
}

func encodeSpace(fragm string, p penalties, styles styled.Style,
	regs *params.TypesettingRegisters) *Khipu {
	//
	T().Debugf("khipukamayuq: encode space with penalites %v", p)
	k := NewKhipu()
	g := spaceglue(regs)
	k.AppendKnot(g).AppendKnot(Penalty(p.p2))
	return k
}

// Currently we do a re-scan of every segment to extract word break opportunities.
// That is obviously not the most efficient way to go about it, as we already scanned
// every input code-point to get here in the first place.
//
// But it is very complicated and hard to reason about segmenting without thinking
// in some kind of hierarchy. Thus, we will approach the problem this way, and in
// a later step interweave UAX segmenting of sentences, words and graphemes into
// one scan. The uax package is able to do that, so it's more about my brain not
// being able to work it out in one go. For now I rather incrementally reduce complexity
// by hierarchy.
//
// The final task will be to re-structure and split this up into co-routines and have
// a monoid on text to produce the khipu. This is limited by at least two constraints:
// First, some UAX segmentations require a O(N) scan of code-points (I did not event start to
// reason about parallelizing the Bidi algorithm, but I'm afraid it will be a nightmare).
// Second, shaping requires front-to-end traversal of code-points as well, and for some
// languages may not even be chunked at whitespace (although some systems make this
// assumption, even a pre-requisite).
//
// For now, our proposition is that paragraphs are the finest level down to which we can
// parallelize things. From paragraphs on we switch to sequential mode.
//
func encodeText(fragm string, item styledItem, env typEnv) *Khipu {
	//
	wordsKhipu := NewKhipu()
	// 1. break fragment into words by UAX#29
	env.pipeline.words.Init(strings.NewReader(fragm))
	pos := item.from
	T().Debugf("################### start word breaker on '%s'", fragm)
	for env.pipeline.words.Next() {
		word := env.pipeline.words.Text()
		T().Debugf("      word = '%s'", word)
		if len(word) == 0 { // should never happen, but be careful not to panic
			continue
		}
		end := pos + uint64(len(word))
		T().Debugf("encode text: word = '%s'", word)
		bidiDir := env.levels.DirectionAt(pos)
		bidiEnd := env.levels.DirectionAt(end - 1)
		if bidiDir != bidiEnd {
			panic("bidi-level changes mid-word")
			// TODO: iterate over word and bidi-level until point of change
			// or: have an API for this in bidi.ResolvedLevels
		}
		// 2. configure shaper
		env.shaper.SetDirection(directionForText(item.styles, bidiDir, env.regs))
		env.shaper.SetScript(scriptForText(item.styles, env.regs))
		env.shaper.SetLanguage(env.regs.S(params.P_LANGUAGE))
		// 3. do NOT hyphenate => leave this to line breaker
		// 4. attach glyph sequences to text boxes
		box := NewTextBox(word, pos)
		//
		styleset := item.styles.(frame.StyleSet)
		box.glyphs = env.shaper.Shape(word, styleset.Font())
		//
		// 5. measure text of glyph sequence
		box.Width, box.Height, box.Depth = box.glyphs.BBox()
		pos = end
		wordsKhipu.AppendKnot(box)
	}
	T().Debugf("###############################################")
	return wordsKhipu
}

func directionForText(styles styled.Style, dir bidi.Direction,
	regs *params.TypesettingRegisters) glyphing.Direction {
	//
	switch dir {
	case bidi.LeftToRight:
		return glyphing.LeftToRight
	case bidi.RightToLeft:
		return glyphing.RightToLeft
	}
	T().Infof("khipukamayuq: vertical text directions not yet implemented")
	return glyphing.LeftToRight
}

func scriptForText(styles styled.Style, regs *params.TypesettingRegisters) glyphing.ScriptID {
	scr := regs.S(params.P_SCRIPT)
	if scr == "" {
		return glyphing.Latin
	}
	return glyphing.ScriptByName(scr)
}

// KnotEncode transforms an input text into a khipu.
func KnotEncode(text io.Reader, startpos uint64, pipeline *TypesettingPipeline,
	regs *params.TypesettingRegisters) *Khipu {
	//
	if regs == nil {
		regs = params.NewTypesettingRegisters()
	}
	pipeline = PrepareTypesettingPipeline(text, pipeline)
	textpos := startpos
	khipu := NewKhipu()
	seg := pipeline.segmenter
	for seg.Next() {
		fragment := seg.Text()
		p := penlty(seg.Penalties())
		T().Debugf("next segment = '%s'\twith penalties %d|%d", fragment, p.p1, p.p2)
		k := createPartialKhipuFromSegment(seg, textpos, pipeline, regs)
		if regs.N(params.P_MINHYPHENLENGTH) < dimen.Infinity {
			HyphenateTextBoxes(k, pipeline, regs)
		}
		khipu.AppendKhipu(k)
	}
	T().Infof("resulting khipu = %s", khipu)
	return khipu
}

// Call this for creating a sub-khipu from a segment. The fist parameter
// is a segmenter which already has detected a segment, i.e. seg.Next()
// has been called successfully.
//
// Calls to createPartialKhipuFromSegment will panic if one of its
// arguments is invalid.
//
// Returns a khipu consisting of text-boxes, glues and penalties.
func createPartialKhipuFromSegment(seg *segment.Segmenter, textpos uint64, pipeline *TypesettingPipeline,
	regs *params.TypesettingRegisters) *Khipu {
	//
	khipu := NewKhipu()
	p := penlty(seg.Penalties())
	T().Errorf("CREATE PARITAL KHIPU, PENALTIES=%d|%d", p.p1, p.p2)
	if p.canWrapLine() { // broken by primary breaker
		// fragment is terminated by possible line wrap opportunity
		if p.breaksAtSpace() { // broken by secondary breaker, too
			if isspace(seg.Text()) {
				g := spaceglue(regs)
				khipu.AppendKnot(g).AppendKnot(Penalty(p.p2))
			} else {
				b := NewTextBox(seg.Text(), textpos)
				khipu.AppendKnot(b).AppendKnot(Penalty(dimen.Infinity))
			}
		} else { // identified as a possible line break, but no space
			// insert explicit discretionary '\-' penalty
			b := NewTextBox(seg.Text(), textpos)
			pen := Penalty(regs.N(params.P_HYPHENPENALTY))
			khipu.AppendKnot(b).AppendKnot(pen)
		}
	} else { // segment is broken by secondary breaker
		// fragment is start or end of a span of whitespace
		if isspace(seg.Text()) {
			T().Errorf("BROKEN BY SECONDARY BREAKER: WHITESPACE")
			// close a span of whitespace
			g := spaceglue(regs)
			pen := Penalty(p.p2)
			khipu.AppendKnot(g).AppendKnot(pen)
		} else {
			T().Errorf("BROKEN BY SECONDARY BREAKER: TEXT_BOX")
			// close a text box which is not a possible line wrap position
			b := NewTextBox(seg.Text(), textpos)
			pen := Penalty(dimen.Infinity)
			khipu.AppendKnot(b).AppendKnot(pen)
		}
	}
	return khipu
}

// HyphenateTextBoxes hypenates all the words in a khipu.
// Words are contained inside TextBox knots.
//
// Hyphenation is governed by the typesetting registers.
// If regs is nil, no hyphenation is done.
func HyphenateTextBoxes(khipu *Khipu, pipeline *TypesettingPipeline,
	regs *params.TypesettingRegisters) {
	//
	if regs == nil || khipu == nil {
		return
	}
	k := make([]Knot, 0, khipu.Length())
	iterator := NewCursor(khipu)
	for iterator.Next() {
		if iterator.Knot().Type() != KTTextBox { // can only hyphenate text knots
			k = append(k, iterator.Knot())
			continue
		}
		T().Debugf("knot = %v | %v", iterator.Knot(), iterator.Knot())
		textbox := iterator.AsTextBox()
		textpos := textbox.Position
		text := textbox.text
		pipeline.words.Init(strings.NewReader(text))
		for pipeline.words.Next() {
			word := pipeline.words.Text()
			T().Debugf("   word = '%s'", word)
			var syllables []string
			isHyphenated := false
			if len(word) >= regs.N(params.P_MINHYPHENLENGTH) {
				if syllables, isHyphenated = HyphenateWord(word, regs); isHyphenated {
					hyphen := NewKnot(KTDiscretionary)
					pos := textpos
					for _, sy := range syllables[:len(syllables)-1] {
						k = append(k, NewTextBox(sy, pos))
						k = append(k, hyphen)
						pos += uint64(len(sy))
					}
					k = append(k, NewTextBox(syllables[len(syllables)-1], pos))
				}
			}
			if !isHyphenated {
				if word == text {
					k = append(k, iterator.Knot())
				} else {
					k = append(k, NewTextBox(word, textpos))
				}
			}
			textpos += uint64(len(word))
		}
	}
	khipu.knots = k
}

// PrepareTypesettingPipeline checks if a typesetting pipeline is correctly
// initialized and creates a new one if is is invalid.
//
// We use a uax14.LineWrapper as the primary breaker and
// use a segment.SimpleWordBreaker to extract spans of whitespace.
// For the inner loop we use a uax29.WordBreaker.
// This is a default configuration adequate for western languages.
func PrepareTypesettingPipeline(text io.Reader, pipeline *TypesettingPipeline) *TypesettingPipeline {
	// wrap a normalization-reader around the input
	if pipeline == nil {
		pipeline = &TypesettingPipeline{}
	}
	pipeline.input = bufio.NewReader(norm.NFC.Reader(text))
	if pipeline.segmenter == nil {
		// pipeline.linewrap = uax14.NewLineWrap()
		// pipeline.segmenter = segment.NewSegmenter(pipeline.linewrap, segment.NewSimpleWordBreaker())
		pipeline.segmenter = segment.NewSegmenter()
		pipeline.segmenter.Init(pipeline.input)
		pipeline.wordbreaker = uax29.NewWordBreaker(1)
		pipeline.words = segment.NewSegmenter(pipeline.wordbreaker)
		pipeline.words.BreakOnZero(true, false)
	}
	return pipeline
}

// HyphenateWord hyphenates a single word.
func HyphenateWord(word string, regs *params.TypesettingRegisters) ([]string, bool) {
	dict := locate.Dictionary(regs.S(params.P_LANGUAGE))
	ok := false
	if dict == nil {
		panic("TODO not yet implemented: find dictionnary for language")
	}
	T().Debugf("   will try to hyphenate word")
	splitWord := dict.Hyphenate(word)
	if len(splitWord) > 1 {
		ok = true
	}
	T().Debugf("   %v", splitWord)
	return splitWord, ok
}

// ---------------------------------------------------------------------------

type penalties struct {
	p1, p2 int
}

func penlty(p1, p2 int) penalties {
	return penalties{p1, p2}
}

func (p penalties) primaryBreak() bool {
	return p.p1 < uax.InfinitePenalty
}

func (p penalties) canWrapLine() bool {
	return p.p1 < uax.InfinitePenalty
}

func (p penalties) breaksAtSpace() bool {
	return p.p2 < uax.InfinitePenalty
}

func isspace(text string) bool {
	if len(text) == 0 {
		return false
	}
	r, width := utf8.DecodeRuneInString(text)
	if width == 0 || r == utf8.RuneError {
		return false
	}
	return unicode.IsSpace(r)
}

func spaceglue(regs *params.TypesettingRegisters) Glue {
	return NewGlue(5*dimen.PT, 1*dimen.PT, 2*dimen.PT)
}
