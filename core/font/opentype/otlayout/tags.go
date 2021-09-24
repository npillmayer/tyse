package otlayout

import "github.com/npillmayer/tyse/core/font/opentype/ot"

// LayoutTagType denotes the type an OpenType layout tag as registered here:
// https://docs.microsoft.com/en-us/typography/opentype/spec/ttoreg
type LayoutTagType uint8

const (
	GSubFeatureType LayoutTagType = 1
	GPosFeatureType LayoutTagType = 2
	ScriptType      LayoutTagType = 3
	LanguageType    LayoutTagType = 4
	BaselineType    LayoutTagType = 5
)

// RegisteredFeatureTags is a list of all the layout features currently (Dec 2020)
// registered at
// https://docs.microsoft.com/en-us/typography/opentype/spec/featurelist.
//
// Please note: features 'cv01'–'cv99' and features 'ss01'–'ss20' are not
// listed here, but will be found in a font by other means.
// Also, some features are not strictly required to exclusively be in GSUB or GPOS,
// but allow for implementations in either table. This will be taken into account
// by `FontFeature(…)` as well.
//
var RegisteredFeatureTags = map[ot.Tag]LayoutTagType{
	ot.T("aalt"): GSubFeatureType, // Access All Alternates
	ot.T("abvf"): GSubFeatureType, // Above-base Forms
	ot.T("abvm"): GPosFeatureType, // Above-base Mark Positioning
	ot.T("abvs"): GSubFeatureType, // Above-base Substitutions
	ot.T("afrc"): GSubFeatureType, // Alternative Fractions
	ot.T("akhn"): GSubFeatureType, // Akhands
	ot.T("blwf"): GSubFeatureType, // Below-base Forms
	ot.T("blwm"): GPosFeatureType, // Below-base Mark Positioning
	ot.T("blws"): GSubFeatureType, // Below-base Substitutions
	ot.T("calt"): GSubFeatureType, // Contextual Alternates
	ot.T("case"): GSubFeatureType, // Case-Sensitive Forms
	ot.T("ccmp"): GSubFeatureType, // Glyph Composition / Decomposition
	ot.T("cfar"): GSubFeatureType, // Conjunct Form After Ro
	ot.T("chws"): GSubFeatureType, // Contextual Half-width Spacing
	ot.T("cjct"): GSubFeatureType, // Conjunct Forms
	ot.T("clig"): GSubFeatureType, // Contextual Ligatures
	ot.T("cpct"): GPosFeatureType, // Centered CJK Punctuation
	ot.T("cpsp"): GPosFeatureType, // Capital Spacing
	ot.T("cswh"): GSubFeatureType, // Contextual Swash
	ot.T("curs"): GPosFeatureType, // Cursive Positioning
	// cv01 - cv99
	ot.T("c2pc"): GSubFeatureType, // Petite Capitals From Capitals
	ot.T("c2sc"): GSubFeatureType, // Small Capitals From Capitals
	ot.T("dist"): GPosFeatureType, // Distances
	ot.T("dlig"): GSubFeatureType, // Discretionary Ligatures
	ot.T("dnom"): GSubFeatureType, // Denominators
	ot.T("dtls"): GSubFeatureType, // Dotless Forms
	ot.T("expt"): GSubFeatureType, // Expert Forms
	ot.T("falt"): GSubFeatureType, // Final Glyph on Line Alternates
	ot.T("fin2"): GSubFeatureType, // Terminal Forms #2
	ot.T("fin3"): GSubFeatureType, // Terminal Forms #3
	ot.T("fina"): GSubFeatureType, // Terminal Forms
	ot.T("flac"): GSubFeatureType, // Flattened accent forms
	ot.T("frac"): GSubFeatureType, // Fractions
	ot.T("fwid"): GSubFeatureType, // Full Widths
	ot.T("half"): GSubFeatureType, // Half Forms
	ot.T("haln"): GSubFeatureType, // Halant Forms
	ot.T("halt"): GSubFeatureType, // Alternate Half Widths
	ot.T("hist"): GSubFeatureType, // Historical Forms
	ot.T("hkna"): GSubFeatureType, // Horizontal Kana Alternates
	ot.T("hlig"): GSubFeatureType, // Historical Ligatures
	ot.T("hngl"): GSubFeatureType, // Hangul
	ot.T("hojo"): GSubFeatureType, // Hojo Kanji Forms (JIS X 0212-1990 Kanji Forms)
	ot.T("hwid"): GSubFeatureType, // Half Widths  // not strictly GSUB !
	ot.T("init"): GSubFeatureType, // Initial Forms
	ot.T("isol"): GSubFeatureType, // Isolated Forms
	ot.T("ital"): GSubFeatureType, // Italics
	ot.T("jalt"): GSubFeatureType, // Justification Alternates
	ot.T("jp78"): GSubFeatureType, // JIS78 Forms
	ot.T("jp83"): GSubFeatureType, // JIS83 Forms
	ot.T("jp90"): GSubFeatureType, // JIS90 Forms
	ot.T("jp04"): GSubFeatureType, // JIS2004 Forms
	ot.T("kern"): GPosFeatureType, // Kerning
	ot.T("lfbd"): GPosFeatureType, // Left Bounds
	ot.T("liga"): GSubFeatureType, // Standard Ligatures
	ot.T("ljmo"): GSubFeatureType, // Leading Jamo Forms
	ot.T("lnum"): GSubFeatureType, // Lining Figures
	ot.T("locl"): GSubFeatureType, // Localized Forms
	ot.T("ltra"): GSubFeatureType, // Left-to-right alternates
	ot.T("ltrm"): GSubFeatureType, // Left-to-right mirrored forms
	ot.T("mark"): GPosFeatureType, // Mark Positioning
	ot.T("med2"): GSubFeatureType, // Medial Forms #2
	ot.T("medi"): GSubFeatureType, // Medial Forms
	ot.T("mgrk"): GSubFeatureType, // Mathematical Greek
	ot.T("mkmk"): GPosFeatureType, // Mark to Mark Positioning
	ot.T("mset"): GSubFeatureType, // Mark Positioning via Substitution
	ot.T("nalt"): GSubFeatureType, // Alternate Annotation Forms
	ot.T("nlck"): GSubFeatureType, // NLC Kanji Forms
	ot.T("nukt"): GSubFeatureType, // Nukta Forms
	ot.T("numr"): GSubFeatureType, // Numerators
	ot.T("onum"): GSubFeatureType, // Oldstyle Figures
	ot.T("opbd"): GPosFeatureType, // Optical Bounds
	ot.T("ordn"): GSubFeatureType, // Ordinals
	ot.T("ornm"): GSubFeatureType, // Ornaments
	ot.T("palt"): GSubFeatureType, // Proportional Alternate Widths
	ot.T("pcap"): GSubFeatureType, // Petite Capitals
	ot.T("pkna"): GSubFeatureType, // Proportional Kana
	ot.T("pnum"): GSubFeatureType, // Proportional Figures
	ot.T("pref"): GSubFeatureType, // Pre-Base Forms
	ot.T("pres"): GSubFeatureType, // Pre-base Substitutions
	ot.T("pstf"): GSubFeatureType, // Post-base Forms
	ot.T("psts"): GSubFeatureType, // Post-base Substitutions
	ot.T("pwid"): GSubFeatureType, // Proportional Widths
	ot.T("qwid"): GSubFeatureType, // Quarter Widths
	ot.T("rand"): GSubFeatureType, // Randomize
	ot.T("rclt"): GSubFeatureType, // Required Contextual Alternates
	ot.T("rkrf"): GSubFeatureType, // Rakar Forms
	ot.T("rlig"): GSubFeatureType, // Required Ligatures
	ot.T("rphf"): GSubFeatureType, // Reph Forms
	ot.T("rtbd"): GPosFeatureType, // Right Bounds
	ot.T("rtla"): GSubFeatureType, // Right-to-left alternates
	ot.T("rtlm"): GSubFeatureType, // Right-to-left mirrored forms
	ot.T("ruby"): GSubFeatureType, // Ruby Notation Forms
	ot.T("rvrn"): GSubFeatureType, // Required Variation Alternates
	ot.T("salt"): GSubFeatureType, // Stylistic Alternates
	ot.T("sinf"): GSubFeatureType, // Scientific Inferiors
	ot.T("size"): GPosFeatureType, // Optical size
	ot.T("smcp"): GSubFeatureType, // Small Capitals
	ot.T("smpl"): GSubFeatureType, // Simplified Forms
	// ss01 - ss20
	ot.T("ssty"): GSubFeatureType, // Math script style alternates
	ot.T("stch"): GSubFeatureType, // Stretching Glyph Decomposition
	ot.T("subs"): GSubFeatureType, // Subscript
	ot.T("sups"): GSubFeatureType, // Superscript
	ot.T("swsh"): GSubFeatureType, // Swash
	ot.T("titl"): GSubFeatureType, // Titling
	ot.T("tjmo"): GSubFeatureType, // Trailing Jamo Forms
	ot.T("tnam"): GSubFeatureType, // Traditional Name Forms
	ot.T("tnum"): GSubFeatureType, // Tabular Figures
	ot.T("trad"): GSubFeatureType, // Traditional Forms
	ot.T("twid"): GSubFeatureType, // Third Widths   // not strictly GSUB !
	ot.T("unic"): GSubFeatureType, // Unicase
	ot.T("valt"): GSubFeatureType, // Alternate Vertical Metrics
	ot.T("vatu"): GSubFeatureType, // Vattu Variants
	ot.T("vchw"): GPosFeatureType, // Vertical Contextual Half-width Spacing
	ot.T("vert"): GSubFeatureType, // Vertical Writing
	ot.T("vhal"): GSubFeatureType, // Alternate Vertical Half Metrics
	ot.T("vjmo"): GSubFeatureType, // Vowel Jamo Forms
	ot.T("vkna"): GSubFeatureType, // Vertical Kana Alternates
	ot.T("vkrn"): GPosFeatureType, // Vertical Kerning
	ot.T("vpal"): GSubFeatureType, // Proportional Alternate Vertical Metrics
	ot.T("vrt2"): GSubFeatureType, // Vertical Alternates and Rotation
	ot.T("vrtr"): GSubFeatureType, // Vertical Alternates for Rotation
	ot.T("zero"): GSubFeatureType, // Slashed Zero
}
