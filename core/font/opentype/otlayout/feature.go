package otlayout

import (
	"fmt"

	"github.com/npillmayer/tyse/core/font/opentype/ot"
)

// Feature is a type for OpenType layout features.
// From the specification website
// https://docs.microsoft.com/en-us/typography/opentype/spec/featuretags :
//
// “Features provide information about how to use the glyphs in a font to render a script or
// language. For example, an Arabic font might have a feature for substituting initial glyph
// forms, and a Kanji font might have a feature for positioning glyphs vertically. All
// OpenType Layout features define data for glyph substitution, glyph positioning, or both.
//
// Each OpenType Layout feature has a feature tag that identifies its typographic function
// and effects. By examining a feature’s tag, a text-processing client can determine what a
// feature does and decide whether to implement it.”
//
// A feature uses ‘lookups’ to do operations on glyphs. GSUB and GPOS tables store lookups in a
// LookupList, into which Features link by maintaining a list of indices into the LookupList.
// The order of the lookup indices matters.
//
type Feature interface {
	Tag() ot.Tag          // e.g., 'liga'
	Type() LayoutTagType  // GSUB or GPOS ?
	Params() ot.Navigator // parameters for this feature
	LookupCount() int     // number of Lookups for this feature
	LookupIndex(int) int  // get index of lookup #i
}

// feature is the default implementation of Feature. Other, more spezialized Feature
// implementations will build on top of this.
type feature struct {
	typ LayoutTagType
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
	lytTables, err := getLayoutTables(otf) // get GSUB and GPOS table for font otf
	if err != nil {
		return nil, nil, err
	}
	var feats = make([][]Feature, 2)
	if script == 0 {
		script = ot.DFLT
	}
	for i := 0; i < 2; i++ { // collect features from GSUB and GPOS
		t := lytTables[i]
		scr := t.ScriptList.LookupTag(script)
		if scr.IsNull() && script != ot.DFLT {
			scr = t.ScriptList.LookupTag(ot.DFLT)
		}
		if scr.IsNull() {
			trace().Infof("font %s has no feature-links from script %s", otf.F.Fontname, script)
			feats[i] = []Feature{}
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
		if lsys == nil || lsys.IsVoid() {
			lsys = dflt
		}
		if lsys == nil || lsys.IsVoid() {
			return nil, nil, errFontFormat(fmt.Sprintf("font %s has empty LangSys entry for %s",
				otf.F.Fontname, script)) // I am not quite sure if this is really illegal
		}
		trace().Debugf("lsys = %v, |lsys| = %d", lsys.Name(), lsys.List().Len())
		flocs := lsys.List().All()
		feats[i] = make([]Feature, len(flocs))
		for j, loc := range flocs { // iterate over all feature records and wrap them into Go types
			inx := loc.U16(0) // inx is an index into a FeatureList
			feats[i][j] = wrapFeature(t, inx, i)
			if feats[i][j] != nil {
				trace().Debugf("%2d: feat[%v] ", j, feats[i][j].Tag())
			}
		}
	}
	return feats[0], feats[1], nil
}

// wrapFeature creates a Feature type from a NavLocation, which should be
// an underlying feature bytes segment.
// `which` is 0 (GSUB) or 1 (GPOS).
func wrapFeature(t *ot.LayoutTable, inx uint16, which int) Feature {
	if inx == 0xffff {
		return nil // 0xffff denotes an unused mandatory feature slot (see OT spec)
	}
	tag, link := t.FeatureList.Get(int(inx))
	f := feature{
		tag: tag,
		nav: link.Navigate(),
	}
	if which == 0 {
		f.typ = GSubFeatureType
	} else {
		f.typ = GPosFeatureType
	}
	return f
}

// Tag returns the identifying tag of this feature.
func (f feature) Tag() ot.Tag {
	return f.tag
}

// Type returns wether this is a GSUB-feature or a GPOS-feature.
func (f feature) Type() LayoutTagType {
	return f.typ
}

// Params returns the parameters for this feature.
func (f feature) Params() ot.Navigator {
	return f.nav.Link().Navigate()
}

// LookupCount returns the number of lookup entries for a feature.
func (f feature) LookupCount() int {
	return f.nav.List().Len()
}

// LookupIndex gets the index-position of lookup number i.
func (f feature) LookupIndex(i int) int {
	if i < 0 || i >= f.nav.List().Len() {
		return -1
	}
	inx := f.nav.List().Get(i).U16(0)
	return int(inx)
}

// --- Feature application ---------------------------------------------------

// ApplyFeature will apply a feature to one or more glyphs of buffer buf, starting at
// position pos. It will return the position after application of the feature.
//
// If a feature is unsuited for the glyph at pos, ApplyFeature will do nothing and return pos.
//
// Attention: It is a requirement that font otf contains the appropriate layout table (either GSUB or
// GPOS) for the feature. Having the table missing may result in a crash. This should never happen, as
// extracting the feature will have required the layout table in the first place. Presence of the
// layout table is not checked again.
//
func ApplyFeature(otf *ot.Font, feat Feature, buf []ot.GlyphIndex, pos, alt int) (int, bool, []ot.GlyphIndex) {
	if feat == nil { // this is legal for unused mandatory feature slots
		return pos, false, buf
	} else if buf == nil || pos < 0 || pos >= len(buf) {
		trace().Infof("application of font-feature requested for unusable buffer condition")
		return pos, false, buf
	}
	var lytTable *ot.LayoutTable
	if feat.Type() == GSubFeatureType {
		lytTable = &otf.Table(ot.T("GSUB")).Self().AsGSub().LayoutTable
	} else {
		lytTable = &otf.Table(ot.T("GPOS")).Self().AsGPos().LayoutTable
	}
	var applied, ok bool
	for i := 0; i < feat.LookupCount(); i++ { // lookups have to be applied in sequence
		inx := feat.LookupIndex(i)
		lookup := lytTable.LookupList.Navigate(inx)
		pos, ok, buf = applyLookup(&lookup, feat, buf, pos, alt)
		applied = applied || ok
	}
	return pos, applied, buf
}

// To apply a lookup, we have to iterate over the lookup's subtables and call them
// appropriately, respecting different subtable semantics and formats.
// Therefore this function more or less is a large switch to delegate to functions
// implementing a specific subtable logic.
func applyLookup(lookup *ot.Lookup, feat Feature, buf []ot.GlyphIndex, pos, alt int) (int, bool, []ot.GlyphIndex) {
	trace().Debugf("applying lookup '%s'/%d", feat.Tag(), lookup.Type)
	for i := 0; i < int(lookup.SubTableCount) && pos < len(buf); i++ {
		trace().Debugf("-------------------- pos = %d", pos)
		// all subtables have the same lookup subtable type, but may have different formats;
		// except for type = Extension
		sub := lookup.Subtable(i)
		if sub == nil {
			continue
		}
		switch sub.LookupType {
		case 1: // Single Substitution Subtable
			switch sub.Format {
			case 1:
				return gsubLookupType1Fmt1(lookup, sub, buf, pos)
			case 2:
				return gsubLookupType1Fmt2(lookup, sub, buf, pos)
			}
		case 2: // Multiple Substitution Subtable
			return gsubLookupType2Fmt1(lookup, sub, buf, pos)
		case 3: // Alternate Substitution Subtable
			return gsubLookupType3Fmt1(lookup, sub, buf, pos, alt)
		case 4: // Ligature Substitution Subtable
			return gsubLookupType4Fmt1(lookup, sub, buf, pos)
		case 5:
			switch sub.Format {
			case 1:
				return gsubLookupType5Fmt1(lookup, sub, buf, pos)
			case 2:
				return gsubLookupType5Fmt2(lookup, sub, buf, pos)
			case 3:
				return gsubLookupType5Fmt3(lookup, sub, buf, pos)
			}
		case 6:
			switch sub.Format {
			case 1:
				return gsubLookupType6Fmt1(lookup, sub, buf, pos)
			case 2:
				return gsubLookupType6Fmt2(lookup, sub, buf, pos)
			case 3:
				return gsubLookupType6Fmt3(lookup, sub, buf, pos)
			}
		}
		trace().Errorf("unknown GSUB lookup type %d/%d", sub.LookupType, sub.Format)
	}
	return pos, false, buf
}

// GSUB LookupType 1: Single Substitution Subtable
//
// Single substitution (SingleSubst) subtables tell a client to replace a single glyph
// with another glyph. The subtables can be either of two formats. Both formats require
// two distinct sets of glyph indices: one that defines input glyphs (specified in the
// Coverage table), and one that defines the output glyphs.

// GSUB LookupSubtable Type 1 Format 1 calculates the indices of the output glyphs, which
// are not explicitly defined in the subtable. To calculate an output glyph index,
// Format 1 adds a constant delta value to the input glyph index. For the substitutions to
// occur properly, the glyph indices in the input and output ranges must be in the same order.
// This format does not use the Coverage index that is returned from the Coverage table.
//
func gsubLookupType1Fmt1(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	_, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d", buf[pos], ok)
	if !ok {
		return pos, false, buf
	}
	// support is deltaGlyphID: add to original glyph ID to get substitute glyph ID
	delta := lksub.Support.(ot.GlyphIndex)
	trace().Debugf("OT lookup GSUB 1/1: subst %d for %d", buf[pos]+delta, buf[pos])
	buf[pos] = buf[pos] + delta
	return pos + 1, true, buf
}

// GSUB LookupSubtable Type 1 Format 2 provides an array of output glyph indices
// (substituteGlyphIDs) explicitly matched to the input glyph indices specified in the
// Coverage table.
// The substituteGlyphIDs array must contain the same number of glyph indices as the
// Coverage table. To locate the corresponding output glyph index in the substituteGlyphIDs
// array, this format uses the Coverage index returned from the Coverage table.
//
func gsubLookupType1Fmt2(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	inx, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d/%v", buf[pos], inx, ok)
	if !ok {
		return pos, false, buf
	}
	if glyph := lookupGlyph(lksub.Index, inx, false); glyph != 0 {
		trace().Debugf("OT lookup GSUB 1/2: subst %d for %d", glyph, buf[pos])
		buf[pos] = glyph
		return pos + 1, true, buf
	}
	return pos, false, buf
}

// LookupType 2: Multiple Substitution Subtable
//
// A Multiple Substitution (MultipleSubst) subtable replaces a single glyph with more
// than one glyph, as when multiple glyphs replace a single ligature.

// GSUB LookupSubtable Type 2 Format 1 defines a count of offsets in the sequenceOffsets
// array (sequenceCount), and an array of offsets to Sequence tables that define the output
// glyph indices (sequenceOffsets). The Sequence table offsets are ordered by the Coverage
// index of the input glyphs.
// For each input glyph listed in the Coverage table, a Sequence table defines the output
// glyphs. Each Sequence table contains a count of the glyphs in the output glyph sequence
// (glyphCount) and an array of output glyph indices (substituteGlyphIDs).
func gsubLookupType2Fmt1(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	inx, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d/%v", buf[pos], inx, ok)
	if !ok {
		return pos, false, buf
	}
	if glyphs := lookupGlyphs(lksub.Index, inx, true); len(glyphs) != 0 {
		trace().Debugf("OT lookup GSUB 2/1: subst %v for %d", glyphs, buf[pos])
		buf = replaceGlyphs(buf, pos, pos+1, glyphs)
		return pos + len(glyphs), true, buf
	}
	return pos, false, buf
}

// LookupType 3: Alternate Substitution Subtable
//
// An Alternate Substitution (AlternateSubst) subtable identifies any number of aesthetic
// alternatives from which a user can choose a glyph variant to replace the input glyph.
// For example, if a font contains four variants of the ampersand symbol, the 'cmap' table
// will specify the index of one of the four glyphs as the default glyph index, and an
// AlternateSubst subtable will list the indices of the other three glyphs as alternatives.
// A text-processing client would then have the option of replacing the default glyph with
// any of the three alternatives.

// GSUB LookupSubtable Type 3 Format 1: For each glyph, an AlternateSet subtable contains a
// count of the alternative glyphs (glyphCount) and an array of their glyph indices
// (alternateGlyphIDs). Parameter `alt` selects an alternative glyph from this array.
// Having `alt` set to -1 will selected the last alternative glyph from the array.
func gsubLookupType3Fmt1(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos, alt int) (
	int, bool, []ot.GlyphIndex) {
	//
	inx, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d/%v", buf[pos], inx, ok)
	if !ok {
		return pos, false, buf
	}
	if glyphs := lookupGlyphs(lksub.Index, inx, true); len(glyphs) != 0 {
		if alt < 0 {
			alt = len(glyphs) - 1
		}
		if alt < len(glyphs) {
			trace().Debugf("OT lookup GSUB 3/1: subst %v for %d", glyphs[alt], buf[pos])
			buf[pos] = glyphs[alt]
			return pos + 1, true, buf
		}
	}
	return pos, false, buf
}

// LookupType 4: Ligature Substitution Subtable
//
// A Ligature Substitution (LigatureSubst) subtable identifies ligature substitutions where
// a single glyph replaces multiple glyphs. One LigatureSubst subtable can specify any number
// of ligature substitutions.

// GSUB LookupSubtable Type 4 Format 1 receives a sequence of glyphs and outputs a
// single glyph replacing the sequence. The Coverage table specifies only the index of the
// first glyph component of each ligature set.
//
// As this is a multi-lookup algorithm, calling gsubLookupType4Fmt1 will return a
// NavLocation which is a LigatureSet, i.e. a list of records of unequal lengths.
//
func gsubLookupType4Fmt1(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	inx, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d/%v", buf[pos], inx, ok)
	if !ok {
		return pos, false, buf
	}
	ligatureSet, err := lksub.Index.Get(inx, false)
	if err != nil || ligatureSet.Size() < 2 {
		return pos, false, buf
	}
	ligCount := ligatureSet.U16(0)
	if ligatureSet.Size() < int(2+ligCount*2) { // must have room for count and u16offset[count]
		return pos, false, buf
	}
	for i := 0; i < int(ligCount); i++ { // iterate over every ligature record in a ligature table
		ligpos := int(ligatureSet.U16(2 + i*2)) // jump to start of ligature record
		if ligatureSet.Size() < ligpos+6 {
			return pos, false, buf
		}
		// Ligature table (glyph components for one ligature):
		// uint16 |  ligatureGlyph                       |  glyph ID of ligature to substitute
		// uint16 |  componentCount                      |  Number of components in the ligature
		// uint16 |  componentGlyphIDs[componentCount-1] |  Array of component glyph IDs
		componentCount := int(ligatureSet.U16(ligpos + 2))
		if componentCount == 0 || componentCount > 10 { // 10 is arbitrary, just to be careful
			continue
		}
		componentGlyphs := ligatureSet.Slice(ligpos+4, ligpos+4+(componentCount-1)*2).Glyphs()
		trace().Debugf("%d component glyphs of ligature: %d %v", componentCount, buf[pos], componentGlyphs)
		// now we know that buf[pos] has matched the first glyph of the component pattern and
		// we will have to match buf[pos+1, ...] to the remaining componentGlyphs
		match := true
		for i, g := range componentGlyphs {
			if pos+i+1 >= len(buf) || g != buf[pos+i+1] {
				match = false
				break
			}
		}
		if match {
			ligatureGlyph := ot.GlyphIndex(ligatureSet.U16(ligpos))
			buf = replaceGlyphs(buf, pos, pos+componentCount, []ot.GlyphIndex{ligatureGlyph})
			trace().Debugf("after application of ligature buf = %v", buf[pos:pos+1])
			return pos + 1, true, buf
		}
	}
	return pos, false, buf
}

// LookupType 5: Contextual Substitution
//
// GSUB type 5 format 1 subtables (and GPOS type 7 format 1 subtables) define input sequences in terms of
// specific glyph IDs. Several sequences may be specified, but each is specified using glyph IDs.
//
// The first glyphs for the sequences are specified in a Coverage table. The remaining glyphs in each
// sequence are defined in SequenceRule tables—one for each sequence. If multiple sequences start with
// the same glyph, that glyph ID must be listed once in the Coverage table, and the corresponding sequence
// rules are aggregated using a SequenceRuleSet table—one for each initial glyph specified in the
// Coverage table.
//
// When evaluating a SequenceContextFormat1 subtable for a given position in a glyph sequence, the client
// searches for the current glyph in the Coverage table. If found, the corresponding SequenceRuleSet
// table is retrieved, and the SequenceRule tables for that set are examined to see if the current glyph
// sequence matches any of the sequence rules. The first matching rule subtable is used.
//
func gsubLookupType5Fmt1(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	inx, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d/%v", buf[pos], inx, ok)
	if !ok {
		return pos, false, buf
	}
	ruleSet, err := lksub.Index.Get(inx, false)
	if err != nil || inx*2 >= ruleSet.Size() { // extra coverage glyphs or extra sequence rule sets are ignored
		return pos, false, buf
	}
	// SequenceRuleSet table – all contexts beginning with the same glyph:
	// uint16   | seqRuleCount                 | Number of SequenceRule tables
	// Offset16 | seqRuleOffsets[posRuleCount] | Array of offsets to SequenceRule tables, from
	//                                           beginning of the SequenceRuleSet table
	seqRuleCnt := ruleSet.U16(0)
	if inx >= int(seqRuleCnt) { // extra coverage glyphs or extra sequence rule sets are ignored
	}
	for i := uint16(0); i < seqRuleCnt; i++ {
		/*
			seqRule, err := ruleSet.Index.Get(i, false) // TODO wrap in array, how? forgot
			if err != nil {
				trace().Debugf("cannot read sequence rule #%d", i)
				return pos, false, buf // ill-formed type 5
			}
		*/
		// SequenceRule table:
		// uint16 | glyphCount                  | Number of glyphs in the input glyph sequence
		// uint16 | seqLookupCount              | Number of SequenceLookupRecords
		// uint16 | inputSequence[glyphCount-1] | Array of input glyph IDs—starting with the second glyph
		// SequenceLookupRecord | seqLookupRecords[seqLookupCount] | Array of Sequence lookup records
		/*
			inputSequence := seqRule.U16(0) // TODO
		*/
	}
	panic("TODO 5/1")
	// return pos, false, buf
}

func gsubLookupType5Fmt2(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	inx, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d/%v", buf[pos], inx, ok)
	if !ok {
		return pos, false, buf
	}
	if lksub.Support == nil {
		trace().Errorf("expected SequenceContext|ClassDefs in field 'Support', is nil")
		return pos, false, buf
	}
	ctx, ok := lksub.Support.(*ot.SequenceContext)
	if !ok {
		trace().Errorf("expected SequenceContext|ClassDefs in field 'Support', type error")
		return pos, false, buf
	}
	trace().Debugf("GSUB lookup type 5|2 has %d ClassDefs", len(ctx.ClassDefs))
	for i, glyph := range buf {
		for j, cdef := range ctx.ClassDefs {
			trace().Debugf(" checking class def #%d for glyph #%d = %d", j, i, glyph)
			clz := cdef.Lookup(glyph)
			trace().Debugf(" class def #%d -> class ID %d", j, clz)
		}
	}
	// => see ot.go: func (lksub LookupSubtable) SequenceRule(b fontBinSegm) sequenceRule
	seqRuleSetCount := lksub.Index.Size()
	trace().Debugf("found %d seq rule sets for SequenceContext Type 5-2", seqRuleSetCount)
	for i := 0; i < seqRuleSetCount; i++ {
		trace().Debugf("=== Rule Set =====================")
		if ruleSetLoc, err := lksub.Index.Get(i, false); err == nil {
			// Index[i] may be 0 => no rule set
			if ruleSetLoc.Size() == 0 {
				trace().Debugf("rule set #%d is empty", i)
				continue
			}
			ruleSet := ot.ParseVarArray(ruleSetLoc, 0, 2, "SequenceRuleSet")
			trace().Debugf("rule set #%d has %d rules in it", i, ruleSet.Size())
			for j := 0; j < ruleSet.Size(); j++ {
				trace().Debugf("--- Rule -------------------------")
				trace().Debugf("checking rule position #%d = %d", j, int(ruleSetLoc.U16(j*2+2)))
				if ruleData, err := ruleSet.Get(j, false); err == nil {
					trace().Debugf("rule data = %v", ruleData.Bytes()[:20])
					glyphCount := ruleData.U16(0)
					trace().Debugf("rule #%d has glyph count = %d", j, glyphCount)
					//rule := ot.ParseVarArray(ruleData, 2, 2+(int(glyphCount)-1)*2, "ClassSequenceRule")
					rule := ot.ParseList(ruleData.Bytes()[4+(int(glyphCount)-1)*2:], int(ruleData.U16(2)), 4)
					//ot.ParseVarArray(ruleData, 2, 2+(int(glyphCount)-1)*2, "ClassSequenceRule")
				} else {
					trace().Errorf("rule #%d: %s", j, err.Error())
					continue
				}
			}
		} else {
			trace().Errorf("rule set #%d: %s", i, err.Error())
			continue
		}
	}
	panic("TODO 5/2")
	// return pos, false, buf
}

func gsubLookupType5Fmt3(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	inx, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d/%v", buf[pos], inx, ok)
	if !ok {
		return pos, false, buf
	}
	panic("TODO 5/3")
	// return pos, false, buf
}

func gsubLookupType6Fmt1(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	inx, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d/%v", buf[pos], inx, ok)
	if !ok {
		return pos, false, buf
	}
	panic("TODO 6/1")
	// return pos, false, buf
}

func gsubLookupType6Fmt2(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	inx, ok := lksub.Coverage.GlyphRange.Match(buf[pos])
	trace().Debugf("coverage of glyph ID %d is %d/%v", buf[pos], inx, ok)
	if !ok {
		return pos, false, buf
	}
	panic("TODO 6/2")
	// return pos, false, buf
}

// Chained Sequence Context Format 3: coverage-based glyph contexts
// GSUB type 6 format 3 subtables and GPOS type 6 format 3 subtables define input sequences patterns, as
// well as chained backtrack and lookahead sequence patterns, in terms of sets of glyph defined using
// Coverage tables.
// The ChainedSequenceContextFormat3 table specifies exactly one input sequence pattern. It has three
// arrays of offsets to coverage tables: one for the input sequence pattern, one for the backtrack
// sequence pattern, and one for the lookahead sequence pattern. For each array, the offsets correspond,
// in order, to the positions in the sequence pattern.
func gsubLookupType6Fmt3(l *ot.Lookup, lksub *ot.LookupSubtable, buf []ot.GlyphIndex, pos int) (
	int, bool, []ot.GlyphIndex) {
	//
	seqctx, ok := lksub.Support.(*ot.SequenceContext)
	if !ok || len(seqctx.InputCoverage) == 0 {
		return pos, false, buf
	}
	for i, cov := range seqctx.InputCoverage {
		if pos+i >= len(buf) {
			return pos, false, buf
		}
		inx, ok := cov.GlyphRange.Match(buf[pos+i])
		trace().Debugf("input coverage of glyph ID %d is %d/%v", buf[pos+i], inx, ok)
		if !ok {
			return pos, false, buf
		}
	}
	for i, cov := range seqctx.BacktrackCoverage {
		if pos-i-1 < 0 {
			return pos, false, buf
		}
		inx, ok := cov.GlyphRange.Match(buf[pos+i])
		trace().Debugf("backtrack coverage of glyph ID %d is %d/%v", buf[pos-i-1], inx, ok)
		if !ok {
			return pos, false, buf
		}
	}
	panic("TODO 6/3")
	//return pos, false, buf
}

// --- Helpers ---------------------------------------------------------------

func replaceGlyphs(buf []ot.GlyphIndex, from, to int, glyphs []ot.GlyphIndex) []ot.GlyphIndex {
	if to <= from {
		return buf
	}
	diff := len(glyphs) - (to - from) // difference in length between old and new
	for diff > len(nullGlyphs) {      // this should never happen
		nullGlyphs = append(nullGlyphs, nullGlyphs...)
	}
	if diff > 0 { // if new glyph sequence is longer than old one => create space
		buf = append(buf, nullGlyphs[:diff]...)
	}
	copy(buf[to+diff:], buf[to:])
	if diff < 0 {
		buf = buf[:len(buf)+diff]
	}
	copy(buf[from:from+len(glyphs)], glyphs)
	return buf
}

var nullGlyphs = []ot.GlyphIndex{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// lookupGlyph is a small helper which looks up an index for a glyph (previously
// returned from a coverage table), checks for errors, and returns the resulting bytes.
// TODO check that this is inlined by the compiler.
func lookupGlyph(index ot.VarArray, ginx int, deep bool) ot.GlyphIndex {
	outglyph, err := index.Get(ginx, deep)
	if err != nil {
		return 0
	}
	return ot.GlyphIndex(outglyph.U16(0))
}

// lookupGlyphs is a small helper which looks up an index for a glyph (previously
// returned from a coverage table), checks for errors, and returns the resulting glyphs.
func lookupGlyphs(index ot.VarArray, ginx int, deep bool) []ot.GlyphIndex {
	outglyphs, err := index.Get(ginx, deep)
	if err != nil {
		return []ot.GlyphIndex{}
	}
	return outglyphs.Glyphs()
}

// get GSUB and GPOS from a font safely
func getLayoutTables(otf *ot.Font) ([]*ot.LayoutTable, error) {
	var table ot.Table
	var lytt = make([]*ot.LayoutTable, 2)
	if table = otf.Table(ot.T("GSUB")); table == nil {
		return nil, errFontFormat(fmt.Sprintf("font %s has no GSUB table", otf.F.Fontname))
	}
	lytt[0] = &table.Self().AsGSub().LayoutTable
	if table = otf.Table(ot.T("GPOS")); table == nil {
		return nil, errFontFormat(fmt.Sprintf("font %s has no GPOS table", otf.F.Fontname))
	}
	lytt[1] = &table.Self().AsGPos().LayoutTable
	return lytt, nil
}

// check if we recognize a feature tag
func identifyFeatureTag(tag ot.Tag) (LayoutTagType, error) {
	if tag&0xffff0000 == ot.T("cv__")&0xffff0000 { // cv00 - cv99
		return GSubFeatureType, nil
	}
	if tag&0xffff0000 == ot.T("ss__")&0xffff0000 { // ss00 - ss20
		return GSubFeatureType, nil
	}
	typ, ok := RegisteredFeatureTags[tag]
	if !ok {
		return 0, errFontFormat(fmt.Sprintf("feature '%s' seems not to be registered", tag))
	}
	return typ, nil
}
