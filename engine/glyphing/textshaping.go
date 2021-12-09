package glyphing

import (
	"fmt"
	"io"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/core/font/opentype"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"golang.org/x/text/language"
)

// Direction is the direction to typeset text in.
type Direction int

// Direction to typeset text in.
//
//go:generate stringer -type=Direction
const (
	LeftToRight Direction = iota
	RightToLeft           = 1
	TopToBottom           = 2
	BottomToTop           = 3
)

// A ShapedGlyph lives in design space (result from the shaper, which lives in design space
// as well, at least its interface).
type ShapedGlyph struct {
	ClusterID  int                       // position of code-point(s) for this glyph in original string
	XAdvance   dimen.DU                  // advance after glyph has been set, in design units
	YAdvance   dimen.DU                  //
	XOffset    dimen.DU                  // position of anchor dot for glyph, in design units
	YOffset    dimen.DU                  //
	RawMetrics opentype.GlyphMetricsInfo // metrics in font units
	GID        ot.GlyphIndex             // glyph index within font
	CodePoint  rune                      // code-point of first rune to produce this glyph
}

func (g ShapedGlyph) String() string {
	return fmt.Sprintf("(GID=%d, advance=%s)", g.GID, g.XOffset)
}

// A Shaper creates a sequence of glyphs from a sequence of
// Unicode code-points. Glyphs are taken from a font, given in a specific point-size.
//
// Clients may provide additional information in Params, as well as
// textual context ([2][]rune).
//
type Shaper interface {
	Shape(io.RuneReader, []ShapedGlyph, [][]rune, Params) (GlyphSequence, error)
}

// Params collects shaping parameters.
type Params struct {
	Font      *font.TypeCase  // use a font at a given point-size
	Direction Direction       // writing direction
	Script    language.Script // 4-letter ISO 15924 script identifier
	Language  language.Tag    // BCP 47 language tag
	Features  []FeatureRange  // OpenType features to apply
}

// FeatureRange tells a shaper to turn a certain OpenType feature on or off for a
// run of code-points.
type FeatureRange struct {
	Feature    ot.Tag // 4-letter feature tag
	Arg        int    // optional argument for this feature
	On         bool   // turn it on or off?
	Start, End int    // position of code-points to apply feature for
}

// GlyphSequence contains a sequence of shaped glyphs.
type GlyphSequence struct {
	Glyphs  []ShapedGlyph // resulting sequence of glyphs
	W, H, D dimen.DU      // width, height, depth of bounding box
}

func (seq GlyphSequence) BoundingBox() (w dimen.DU, h dimen.DU, d dimen.DU) {
	return seq.W, seq.H, seq.D
}
