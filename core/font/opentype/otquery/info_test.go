package otquery

import (
	"testing"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"github.com/stretchr/testify/suite"
)

// --- Test Suite Preparation ------------------------------------------------

type InfoTestEnviron struct {
	suite.Suite
	otf *ot.Font
}

// listen for 'go test' command --> run test methods
func TestInfoFunctions(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	suite.Run(t, new(InfoTestEnviron))
}

// run once, before test suite methods
func (env *InfoTestEnviron) SetupSuite() {
	env.T().Log("Setting up test suite")
	tracing.Select("tyse.fonts").SetTraceLevel(tracing.LevelError)
	env.otf = loadLocalFont(env.T(), "Calibri.ttf")
	tracing.Select("tyse.fonts").SetTraceLevel(tracing.LevelInfo)
}

// run once, after test suite methods
func (env *InfoTestEnviron) TearDownSuite() {
	env.T().Log("Tearing down test suite")
}

// --- Tests -----------------------------------------------------------------

func (env *InfoTestEnviron) TestFontTypeInfo() {
	fti := FontType(env.otf)
	env.Equal("TrueType", fti, "expected font type of test font to be TrueType")
}

func (env *InfoTestEnviron) TestGeneralInfo() {
	info := NameInfo(env.otf, ot.DFLT)
	env.T().Logf("info = %v", info)
	fam, ok := info["family"]
	env.Require().True(ok, "font familiy identifier not found in font info")
	env.Equal("Calibri", fam, "expected font family name 'Calibri'")
}

func (env *InfoTestEnviron) TestLayoutInfo() {
	layouts := LayoutTables(env.otf)
	env.T().Logf("test font layout tables: %v", layouts)
	required := []string{"GDEF", "GSUB", "GPOS"}
	for _, reqt := range required {
		env.Contains(layouts, reqt, "expected test font to contain required table %s", reqt)
	}
}

func (env *InfoTestEnviron) TestReverseLookup() {
	r := CodePointForGlyph(env.otf, 4)
	env.Equal('A', r, "expected code-point to be %#U, is %#U", 'A', r)
}

func (env *InfoTestEnviron) TestGlyphClasses() {
	clz := ClassesForGlyph(env.otf, 4) // 4 = 'A'
	env.Equal(1, clz.Class, "expected class of 'A' to be 1, is %d", clz.Class)
}

// --- Helpers ----------------------------------------------------------

/*
func loadCalibri(t *testing.T) *ot.Font {
	f := loadTestFont(t, "calibri")
	otf, err := ot.Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	return otf
}

func loadTestFont(t *testing.T, pattern string) *ot.Font {
	otf := &ot.Font{}
	if pattern == "fallback" {
		otf.F = font.FallbackFont()
	} else {
		conf := testconfig.Conf{
			"fontconfig": "/usr/local/bin/fc-list",
			"app-key":    "tyse-test",
		}
		loader := resources.ResolveTypeCase(conf, pattern, font.StyleNormal, font.WeightNormal, 10.0)
		tyc, err := loader.TypeCase()
		if err != nil {
			t.Fatal(err)
		}
		otf.F = tyc.ScalableFontParent()
	}
	t.Logf("loaded font = %s", otf.F.Fontname)
	return otf
}
*/
