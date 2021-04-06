package otlyt

import (
	"fmt"

	"github.com/npillmayer/tyse/core/font/ot"
)

// Feature is a type for OpenType layout features.
// From the specification Website
// https://docs.microsoft.com/en-us/typography/opentype/spec/featuretags :
//
// Features provide information about how to use the glyphs in a font to render a script or
// language. For example, an Arabic font might have a feature for substituting initial glyph
// forms, and a Kanji font might have a feature for positioning glyphs vertically. All
// OpenType Layout features define data for glyph substitution, glyph positioning, or both.
//
// Each OpenType Layout feature has a feature tag that identifies its typographic function
// and effects. By examining a feature’s tag, a text-processing client can determine what a
// feature does and decide whether to implement it.
//
type Feature interface {
	Tag() ot.Tag           // e.g., 'liga'
	Apply([]rune, int) int // apply feature to one or more glyphs
}

type feature struct {
	tag ot.Tag
	nav ot.Navigator
}

// FontFeature looks up OpenType layout features in OpenType font otf, i.e. it trys to
// find features in table GSUB as well as in table GPOS.
// In OpenType, features may be specific for script/language combinations, or DFLT.
// Also, some (few) features may have a GSUB part as well as a GPOS part.
// Setting script to 0 will look for a DFLT feature set.
//
// Returns GSUB features, GPOS features and a possible error condition.
// The features at index 0 of each slice are the mandatory features (for a script), and may
// be nil.
//
func FontFeatures(otf *ot.Font, script, lang ot.Tag) ([]Feature, []Feature, error) {
	lytTables, err := getLayoutTables(otf)
	if err != nil {
		return nil, nil, err
	}
	var feats = make([][]Feature, 2)
	if script == 0 {
		script = ot.T("DFLT")
	}
	for i := 0; i < 2; i++ {
		t := lytTables[i]
		scr := t.ScriptList.LookupTag(script)
		if scr.IsNull() && script != ot.T("DFLT") {
			scr = t.ScriptList.LookupTag(ot.T("DFLT"))
		}
		if scr.IsNull() {
			trace().Infof("font %s has no feature-links from script %s", otf.F.Fontname, script)
			continue
		}
		trace().Debugf("found script table for '%s'", script)
		langs := scr.Navigate()
		//trace().Debugf("now at table %s", langs.Name())
		var dflt, lsys ot.Navigator
		dflt = langs.Link().Navigate()
		if lang != 0 {
			if lptr := langs.Map().LookupTag(lang); !lptr.IsNull() {
				lsys = lptr.Navigate()
			}
		}
		trace().Debugf("dflt = %v, |feats| = %d", dflt.Name(), dflt.List().Len())
		if lsys != nil && !lsys.IsVoid() {
			trace().Debugf("lsys = %v", lsys.Name())
		} else {
			lsys = dflt
		}
		flocs := lsys.List().All()
		feats[i] = make([]Feature, len(flocs))
		for j, loc := range flocs {
			inx := loc.U16(0)
			//trace().Debugf("loc[%d] = %d", j, inx)
			feats[i][j] = wrapFeature(t, inx, i)
			trace().Debugf("%2d: feat[%v] ", j, feats[i][j].Tag())
		}
	}
	return feats[0], feats[1], nil
}

// wrapFeature creates a Feature type from a NavLocation, which should be
// an underlying feature bytes segment.
func wrapFeature(t *ot.LayoutTable, inx uint16, which int) Feature {
	if inx == 0xffff {
		return feature{}
	}
	featureList := t.FeatureList
	tag, link := featureList.Get(int(inx))
	return feature{
		tag: tag,
		nav: link.Navigate(),
	}
}

// Tag returns the identifying tag of this feature.
func (f feature) Tag() ot.Tag {
	return f.tag
}

// Apply will apply a feature to one or more glyphs of buffer buf, starting at
// position pos. Will return the position after application of the feature.
//
// If a feature is unsuited for the glyph at pos, Apply will do nothing and return pos.
func (f feature) Apply(buf []rune, pos int) int {
	return 0
}

// get GSUB and GPOS from a font
func getLayoutTables(otf *ot.Font) ([]*ot.LayoutTable, error) {
	var table ot.Table
	var lytt = make([]*ot.LayoutTable, 2, 2)
	if table = otf.Table(ot.T("GSUB")); table == nil {
		return nil, fmt.Errorf("font %s has no GSUB table", otf.F.Fontname)
	}
	lytt[0] = &table.Self().AsGSub().LayoutTable
	if table = otf.Table(ot.T("GPOS")); table == nil {
		return nil, fmt.Errorf("font %s has no GPOS table", otf.F.Fontname)
	}
	lytt[1] = &table.Self().AsGPos().LayoutTable
	return lytt, nil
}

func identifyFeatureTag(tag ot.Tag) (LayoutTagType, error) {
	if tag&0xffff0000 == ot.T("cv__")&0xffff0000 { // cv00 - cv99
		return GSubFeatureType, nil
	}
	if tag&0xffff0000 == ot.T("ss__")&0xffff0000 { // ss00 - ss20
		return GSubFeatureType, nil
	}
	typ, ok := RegisteredFeatureTags[tag]
	if !ok {
		return 0, fmt.Errorf("feature '%s' seems not to be registered", tag)
	}
	return typ, nil
}
