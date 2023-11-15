package otshape

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core/font/opentype"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"github.com/npillmayer/tyse/core/font/opentype/otquery"
	"github.com/stretchr/testify/suite"
	"golang.org/x/image/font"
	xot "golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// --- Test Suite Preparation ------------------------------------------------

type BufferTestEnviron struct {
	suite.Suite
	otf         *ot.Font
	fontMetrics opentype.FontMetricsInfo
	imageName   string
}

// listen for 'go test' command --> run test methods
func TestBufferFunctions(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	suite.Run(t, new(BufferTestEnviron))
}

// run once, before test suite methods
func (env *BufferTestEnviron) SetupSuite() {
	env.T().Log("Setting up test suite")
	tracing.Select("tyse.fonts").SetTraceLevel(tracing.LevelError)
	env.otf = loadLocalFont(env.T(), "Calibri.ttf")
	env.fontMetrics = otquery.FontMetrics(env.otf)
	tracing.Select("tyse.fonts").SetTraceLevel(tracing.LevelDebug)
}

// run once, after test suite methods
func (env *BufferTestEnviron) TearDownSuite() {
	env.T().Log("Tearing down test suite")
}

// run after each test.
// If env.imageName is non-empty, then we will export a proof image of the
// current buffer contents.
func (env *BufferTestEnviron) TearDownTest() {
	if env.imageName != "" {
		// write ../testdata/proofs/<env.imageName>.png
		displayBuffer(env, env.imageName)
	}
	env.imageName = ""
}

// --- Tests -----------------------------------------------------------------

func (env *BufferTestEnviron) TestBufferInitialMapping() {
	data := []struct {
		in     string
		length int
		want   []ot.GlyphIndex
	}{
		{"A", 1, []ot.GlyphIndex{4}},
		{"é", 1, []ot.GlyphIndex{288}},
		{"e\u0301", 1, []ot.GlyphIndex{288}}, // NFD é
		{"Café", 4, []ot.GlyphIndex{18, 258, 296, 288}},
	}
	buf := NewBuffer(16)
	for _, pair := range data {
		n := buf.mapGlyphs(pair.in, env.otf, ot.DFLT, ot.DFLT)
		env.Equal(pair.length, n, "not all codepoints mapped to glyph positions")
		//env.T().Logf("buffer is %v", buf[:n])
		env.Equal(pair.want, buf.Glyphs()[:n])
	}
}

func (env *BufferTestEnviron) TestBufferDraw() {
	env.imageName = "jelly"
}

// --- Helpers ---------------------------------------------------------------

func displayBuffer(env *BufferTestEnviron, basename string) {
	const (
		width        = 800
		height       = 300
		startingDotX = 30
		startingDotY = 220
		size         = float32(190)
		DPI          = 72
	)
	face, err := xot.NewFace(env.otf.F.SFNT, &xot.FaceOptions{
		Size:    190,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	env.Require().NoError(err, "cannot create NewFace from OpenType font")
	dst := image.NewGray(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{image.White}, image.Point{}, draw.Src)
	d := font.Drawer{
		Dst:  dst,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(startingDotX, startingDotY),
	}
	env.T().Logf("The dot is at %v\n", d.Dot)
	d.DrawString("jel")
	env.T().Logf("The dot is at %v\n", d.Dot)
	d.DrawString("ly")
	env.T().Logf("The dot is at %v\n", d.Dot)
	f, err := os.Create(filepath.Join("..", "testdata", "proofs", basename+".png"))
	env.Require().NoError(err, "cannot open/create PNG output file")
	defer f.Close()
	png.Encode(f, d.Dst)
}

func drawGlyph(d font.Drawer, glyph ot.GlyphIndex, advance int) font.Drawer {
	d.Dot = d.Dot.Add(fixed.P(advance, 0))
	return d
}
