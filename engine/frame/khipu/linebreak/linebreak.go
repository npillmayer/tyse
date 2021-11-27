/*
Package linebreak collects types for line-breaking.

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
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. */
package linebreak

// https://quod.lib.umich.edu/j/jep/3336451.0013.105?view=text;rgn=main

import (
	"fmt"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/frame/khipu"
)

// T traces to a global core tracer.
func T() tracing.Trace {
	return gtrace.CoreTracer
}

type Merits int32
type CharPos int64

// ----------------------------------------------------------------------

// Parameters is a collection of configuration parameters for line-breaking.
type Parameters struct {
	Tolerance            Merits     // acceptable demerits
	PreTolerance         Merits     // acceptabale demerits for first (rough) pass
	LinePenalty          Merits     // penalty for an additional line
	HyphenPenalty        Merits     // penalty for hyphenating words
	ExHyphenPenalty      Merits     // penalty for explicit words
	DoubleHyphenDemerits Merits     // demerits for consecutive hyphens
	FinalHyphenDemerits  Merits     // demerits for hyphen in the last line
	EmergencyStretch     dimen.DU   // stretching acceptable when desperate
	LeftSkip             khipu.Glue // glue at left edge of paragraphs
	RightSkip            khipu.Glue // glue at right edge of paragraphs
	ParFillSkip          khipu.Glue // glue at the end of a paragraph
}

// DefaultParameters are the standard line-breaking parameters.
// The promote a tolerant configuration, suitable for almost always finding an
// acceptable set of linebreaks.
var DefaultParameters = &Parameters{
	Tolerance:            5000,
	PreTolerance:         100,
	LinePenalty:          10,
	HyphenPenalty:        50,
	ExHyphenPenalty:      50,
	DoubleHyphenDemerits: 0,
	FinalHyphenDemerits:  50,
	EmergencyStretch:     dimen.DU(dimen.BP * 50),
	LeftSkip:             khipu.NewGlue(0, 0, 0),
	RightSkip:            khipu.NewGlue(0, 0, 0),
	ParFillSkip:          khipu.NewGlue(0, 0, 0),
}

// ----------------------------------------------------------------------

// WSS (width stretch & shrink) is a type to hold an elastic width (of text).
type WSS struct {
	W   dimen.DU
	Min dimen.DU
	Max dimen.DU
}

// Spread returns the width's of an elastic WSS.
func (wss WSS) Spread() (w dimen.DU, min dimen.DU, max dimen.DU) {
	return wss.W, wss.Min, wss.Max
}

// SetFromKnot sets the width's of an elastic WSS from a knot.
func (wss WSS) SetFromKnot(knot khipu.Knot) WSS {
	if knot == nil {
		return wss
	}
	wss.W = knot.W()
	wss.Min = knot.MinW()
	wss.Max = knot.MaxW()
	return wss
}

// Add adds dimensions from a given WSS to wss, returning a new WSS.
func (wss WSS) Add(other WSS) WSS {
	return WSS{
		W:   wss.W + other.W,
		Min: wss.Min + other.Min,
		Max: wss.Max + other.Max,
	}
}

// Subtract subtracts dimensions from a given WSS from wss, returning a new WSS.
func (wss WSS) Subtract(other WSS) WSS {
	return WSS{
		W:   wss.W - other.W,
		Min: wss.Min - other.Min,
		Max: wss.Max - other.Max,
	}
}

// Copy copies a WSS.
func (wss WSS) Copy() WSS {
	return WSS{W: wss.W, Min: wss.Min, Max: wss.Max}
}

func (wss WSS) String() string {
	return fmt.Sprintf("{%.2f < %.2f < %.2f}", wss.Min.Points(), wss.W.Points(), wss.Max.Points())
}

// InfinityDemerits is the worst demerit value possible.
const InfinityDemerits Merits = 10000

// InfinityMerits is the best (most desirable) demerit value possible.
const InfinityMerits Merits = -10000

// CapDemerits caps a demerit value at infinity.
func CapDemerits(d Merits) Merits {
	if d > InfinityDemerits {
		d = InfinityDemerits
	} else if d < InfinityMerits-1000 {
		d = InfinityMerits - 1000
	}
	return d
}

// --- Interfaces -------------------------------------------------------

// Cursor is a type to iterate over a khipu.
type Cursor interface {
	Next() bool
	Knot() khipu.Knot
	Peek() (khipu.Knot, bool)
	Mark() khipu.Mark
	Khipu() *khipu.Khipu
}

// ParShape is a type to return the line length for a given line number.
type ParShape interface {
	LineLength(int32) dimen.DU
}

type rectParShape dimen.DU

func (r rectParShape) LineLength(int32) dimen.DU {
	return dimen.DU(r)
}

// RectangularParShape returns a Parshape for paragraphs of constant line length.
func RectangularParShape(linelen dimen.DU) ParShape {
	return rectParShape(linelen)
}
