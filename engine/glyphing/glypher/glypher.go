package glypher

import (
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/engine/glyphing"
)

// Package intended for a home-grown shaper for easy cases where we can
// afford to not rely on HarfBuzz.

type glypher struct {
	dir    glyphing.Direction
	script glyphing.ScriptID
}

func Instance(dir glyphing.Direction, script glyphing.ScriptID) glyphing.Shaper {
	return &glypher{
		dir:    dir,
		script: script,
	}
}

func (g *glypher) Shape(text string, typecase *font.TypeCase) glyphing.GlyphSequence {
	panic("Glyphing Shape: TODO")
}

func (g *glypher) SetScript(scr glyphing.ScriptID) {
	g.script = scr
}

func (g *glypher) SetDirection(dir glyphing.Direction) {
	g.dir = dir
}

func (g *glypher) SetLanguage(string) {
	//
}
