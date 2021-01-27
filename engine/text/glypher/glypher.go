package glypher

import (
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/engine/text"
)

// Package intended for a home-grown shaper for easy cases where we can
// afford to not rely on HarfBuzz.

type glyphing struct {
	dir    text.TextDirection
	script text.ScriptID
}

func Instance(dir text.TextDirection, script text.ScriptID) text.Shaper {
	return &glyphing{
		dir:    dir,
		script: script,
	}
}

func (g *glyphing) Shape(text string, typecase *font.TypeCase) text.GlyphSequence {
	panic("Glyphing Shape: TODO")
}

func (g *glyphing) SetScript(scr text.ScriptID) {
	g.script = scr
}

func (g *glyphing) SetDirection(dir text.TextDirection) {
	g.dir = dir
}

func (g *glyphing) SetLanguage() {
	//
}
