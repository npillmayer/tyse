package otquery

import (
	"path/filepath"
	"testing"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"github.com/stretchr/testify/suite"
	"golang.org/x/image/font/sfnt"
)

// --- Test Suite Preparation ------------------------------------------------

type MetricsTestEnviron struct {
	suite.Suite
	calibri *ot.Font
}

// listen for 'go test' command --> run test methods
func TestMetricsFunctions(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	suite.Run(t, new(MetricsTestEnviron))
}

// run once, before test suite methods
func (env *MetricsTestEnviron) SetupSuite() {
	env.T().Log("Setting up test suite")
	tracing.Select("tyse.fonts").SetTraceLevel(tracing.LevelError)
	env.calibri = loadLocalFont(env.T(), "Calibri.ttf")
	tracing.Select("tyse.fonts").SetTraceLevel(tracing.LevelInfo)
}

// run once, after test suite methods
func (env *MetricsTestEnviron) TearDownSuite() {
	env.T().Log("Tearing down test suite")
}

// --- Tests -----------------------------------------------------------------

func (env *MetricsTestEnviron) TestGlyphIndex() {
	gid := GlyphIndex(env.calibri, 'A')
	env.Equal(ot.GlyphIndex(4), gid, "expected glyph index of 'A' in test font to be 4")
}

func (env *MetricsTestEnviron) TestGlyphMetrics() {
	gid := GlyphIndex(env.calibri, 'A')
	m := GlyphMetrics(env.calibri, gid)
	env.T().Logf("metrics = %v", m)
	env.Equal(sfnt.Units(1185), m.Advance, "expected font.Advance for 'A' to be 1185 units")
}

func (env *MetricsTestEnviron) TestLanguageMatch() {
	script, lang := FontSupportsScript(env.calibri, ot.T("latn"), ot.T("TRK"))
	env.Equal("latn", script.String(), "expected Latin script in test font")
	env.Equal("TRK ", lang.String(), "expected Turkish language support in test font")
}

// --- Helpers ---------------------------------------------------------------

func loadLocalFont(t *testing.T, fontFileName string) *ot.Font {
	path := filepath.Join("..", "testdata", fontFileName)
	f, err := font.LoadOpenTypeFont(path)
	if err != nil {
		t.Fatalf("cannot load test font %s: %s", fontFileName, err)
	}
	t.Logf("loaded SFNT font = %s", f.Fontname)
	otf, err := ot.Parse(f.Binary)
	if err != nil {
		t.Fatalf("cannot decode test font %s: %s", fontFileName, err)
	}
	otf.F = f
	t.Logf("parsed OpenType font = %s", otf.F.Fontname)
	return otf
}
