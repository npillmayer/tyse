package glypher

import (
	"io"

	"github.com/npillmayer/tyse/engine/glyphing"
	"golang.org/x/text/language"
)

// Package intended for a home-grown shaper for easy cases where we can
// afford to not rely on HarfBuzz.

type glypher struct {
	dir    glyphing.Direction
	script language.Script
	lang   language.Tag
}

func Instance(dir glyphing.Direction, script language.Script, lang language.Tag) glyphing.Shaper {
	return &glypher{
		dir:    dir,
		script: script,
		lang:   lang,
	}
}

func (g *glypher) Shape(io.RuneReader, []glyphing.ShapedGlyph, [][]rune, glyphing.Params) (glyphing.GlyphSequence, error) {
	panic("Glyphing Shape: TODO")
}
