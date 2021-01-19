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

3. Neither the name of this software nor the names of its contributors
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
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/locate"
	params "github.com/npillmayer/tyse/core/parameters"
	"github.com/npillmayer/uax"
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

// KnotEncode transforms an input text into a khipu.
func KnotEncode(text io.Reader, pipeline *TypesettingPipeline, regs *params.TypesettingRegisters) *Khipu {
	if regs == nil {
		regs = params.NewTypesettingRegisters()
	}
	pipeline = PrepareTypesettingPipeline(text, pipeline)
	khipu := NewKhipu()
	seg := pipeline.segmenter
	for seg.Next() {
		fragment := seg.Text()
		p := penlty(seg.Penalties())
		CT().Debugf("next segment = '%s'\twith penalties %d|%d", fragment, p.p1, p.p2)
		k := createPartialKhipuFromSegment(seg, pipeline, regs)
		if regs.N(params.P_MINHYPHENLENGTH) < dimen.Infty {
			HyphenateTextBoxes(k, pipeline, regs)
		}
		khipu.AppendKhipu(k)
	}
	CT().Infof("resulting khipu = %s", khipu)
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
func createPartialKhipuFromSegment(seg *segment.Segmenter, pipeline *TypesettingPipeline,
	regs *params.TypesettingRegisters) *Khipu {
	//
	khipu := NewKhipu()
	p := penlty(seg.Penalties())
	CT().Errorf("CREATE PARITAL KHIPU, PENALTIES=%d|%d", p.p1, p.p2)
	if p.primaryBreak() { // broken by primary breaker
		// fragment is terminated by possible line wrap opportunity
		if p.secondaryBreak() { // broken by secondary breaker, too
			if isspace(seg) {
				g := spaceglue(regs)
				khipu.AppendKnot(g).AppendKnot(Penalty(p.p2))
			} else {
				b := NewTextBox(seg.Text())
				khipu.AppendKnot(b).AppendKnot(Penalty(dimen.Infty))
			}
		} else { // identified as a possible line break, but no space
			// insert explicit discretionary '\-' penalty
			b := NewTextBox(seg.Text())
			pen := Penalty(regs.N(params.P_HYPHENPENALTY))
			khipu.AppendKnot(b).AppendKnot(pen)
		}
	} else { // segment is broken by secondary breaker
		// fragment is start or end of a span of whitespace
		if isspace(seg) {
			CT().Errorf("BROKEN BY SECONDARY BREAKER: WHITESPACE")
			// close a span of whitespace
			g := spaceglue(regs)
			pen := Penalty(p.p2)
			khipu.AppendKnot(g).AppendKnot(pen)
		} else {
			CT().Errorf("BROKEN BY SECONDARY BREAKER: TEXT_BOX")
			// close a text box which is not a possible line wrap position
			b := NewTextBox(seg.Text())
			pen := Penalty(dimen.Infty)
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
		CT().Debugf("knot = %v | %v", iterator.Knot(), iterator.Knot())
		text := iterator.AsTextBox().text
		pipeline.words.Init(strings.NewReader(text))
		for pipeline.words.Next() {
			word := pipeline.words.Text()
			CT().Debugf("   word = '%s'", word)
			var syllables []string
			isHyphenated := false
			if len(word) >= regs.N(params.P_MINHYPHENLENGTH) {
				if syllables, isHyphenated = HyphenateWord(word, regs); isHyphenated {
					hyphen := NewKnot(KTDiscretionary)
					for _, sy := range syllables[:len(syllables)-1] {
						k = append(k, NewTextBox(sy))
						k = append(k, hyphen)
					}
					k = append(k, NewTextBox(syllables[len(syllables)-1]))
				}
			}
			if !isHyphenated {
				if word == text {
					k = append(k, iterator.Knot())
				} else {
					k = append(k, NewTextBox(word))
				}
			}
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
		pipeline.linewrap = uax14.NewLineWrap()
		pipeline.segmenter = segment.NewSegmenter(pipeline.linewrap, segment.NewSimpleWordBreaker())
		pipeline.segmenter.Init(pipeline.input)
		pipeline.wordbreaker = uax29.NewWordBreaker()
		pipeline.words = segment.NewSegmenter(pipeline.wordbreaker)
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
	CT().Debugf("   will try to hyphenate word")
	splitWord := dict.Hyphenate(word)
	if len(splitWord) > 1 {
		ok = true
	}
	CT().Debugf("   %v", splitWord)
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

func (p penalties) secondaryBreak() bool {
	return p.p2 < uax.InfinitePenalty
}

func isspace(seg *segment.Segmenter) bool {
	if len(seg.Text()) == 0 {
		return false
	}
	r, width := utf8.DecodeRuneInString(seg.Text())
	if width == 0 || r == utf8.RuneError {
		return false
	}
	return unicode.IsSpace(r)
}

func spaceglue(regs *params.TypesettingRegisters) Glue {
	return NewGlue(5*dimen.PT, 1*dimen.PT, 2*dimen.PT)
}
