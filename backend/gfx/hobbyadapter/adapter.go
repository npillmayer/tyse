/*
Package hobbyadapter implements a bridge to splines according to J.Hobby.
It is intended to be used on a 'Surface', which in turn will relay to
a concrete graphics implementation (e.g., Canvas or Cairo).

BSD License

Copyright (c) 2017-21, Norbert Pillmayer <norbert@pillmayer.com>

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of Norbert Pillmayer nor the names of its contributors
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
package hobbyadapter

import (
	"math/cmplx"

	"github.com/npillmayer/arithm"
	"github.com/npillmayer/arithm/jhobby"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/backend/gfx"
)

// G traces to the graphics tracer.
func G() tracing.Trace {
	return gtrace.GraphicsTracer
}

// Contour creates an immutable adapter to contours from J.Hobby-splines.
func Contour(path jhobby.HobbyPath, controls jhobby.SplineControls) gfx.DrawableContour {
	pdrw := &pathdrawer{p: path, c: controls}
	if path.IsCycle() {
		pdrw.n = path.N() + 1
	} else {
		pdrw.n = path.N()
	}
	return pdrw
}

// internal type: immutable adapter to contours from cubic splines
type pathdrawer struct {
	p       jhobby.HobbyPath
	c       jhobby.SplineControls
	current int
	n       int
}

// implement interface DrawableContour
func (pdrw *pathdrawer) IsCycle() bool {
	return pdrw.p.IsCycle()
}

// implement interface DrawableContour
func (pdrw *pathdrawer) Start() arithm.Pair {
	G().Debugf("path start at %s", arithm.Pair(pdrw.p.Z(0)))
	pdrw.current = 0
	return pdrw.p.Z(0)
}

// implement interface DrawableContour
func (pdrw *pathdrawer) ToNextKnot() (arithm.Pair, arithm.Pair, arithm.Pair) {
	pdrw.current++
	if pdrw.current >= pdrw.n {
		G().Debugf("path has no more knots")
		return arithm.Origin, arithm.Origin, arithm.Origin
	}
	c1, c2 := pdrw.c.PostControl(pdrw.current-1), pdrw.c.PreControl(pdrw.current%(pdrw.p.N()))
	if pdrw.current < pdrw.n && !cmplx.IsNaN(c1.C()) {
		G().Debugf("path next  at %s", arithm.Pair(pdrw.p.Z(pdrw.current)))
		G().Debugf("     controls %s and %s", arithm.Pair(c1), arithm.Pair(c2))
	} else {
		G().Debugf("path next  at %s", arithm.Pair(pdrw.p.Z(pdrw.current)))
	}
	return pdrw.p.Z(pdrw.current), c1, c2
}
