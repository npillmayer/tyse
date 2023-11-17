package otshaper

import (
	"slices"

	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// see https://unicode.org/iso15924/iso15924-codes.html
var script2opentype = map[string]string{
	"Zzzz": "DFLT", // unknown
	//
	"Arab": "arab", // Arabic
	"Armn": "armn", // Armenian
	"Beng": "bng2", // Bengali
	"Cyrl": "cyrl", // Cyrillic
	"Deva": "dev2", // Devangari
	"Geor": "geor", // Georgian
	"Grek": "grek", // Greek
	"Gujr": "gjr2", // Not gujr
	"Guru": "gur2", // Not guru
	"Hang": "hang", // Hanguli
	"Hans": "hani", // Han (simplified)
	"Hebr": "hebr", // Hebrew
	"Hira": "hira", // Hiragana
	"Knda": "knd2", // Kannada
	"Kana": "kana", // Katakana
	"Laoo": "laoo", // Lao
	"Latn": "latn", // Latin
	// TODO
	"Malayalam":              "mlm2",
	"Oriya":                  "ory2",
	"Tamil":                  "tml2",
	"Telugu":                 "tel2",
	"Thai":                   "thai",
	"Tibetan":                "tibt",
	"Bopomofo":               "bopo",
	"Braille":                "brai",
	"Canadian_Syllabics":     "cans",
	"Cherokee":               "cher",
	"Ethiopic":               "ethi",
	"Khmer":                  "khmr",
	"Mongolian":              "mong",
	"Myanmar":                "mym2", // Not mymr
	"Ogham":                  "ogam",
	"Runic":                  "runr",
	"Sinhala":                "sinh",
	"Syriac":                 "syrc",
	"Thaana":                 "thaa",
	"Yi":                     "yiii",
	"Deseret":                "dsrt",
	"Gothic":                 "goth",
	"Old_Italic":             "ital",
	"Buhid":                  "buhd",
	"Hanunoo":                "hano",
	"Tagalog":                "tglg",
	"Tagbanwa":               "tagb",
	"Cypriot":                "cprt",
	"Limbu":                  "limb",
	"Linear_B":               "linb",
	"Osmanya":                "osma",
	"Shavian":                "shaw",
	"Tai_Le":                 "tale",
	"Ugaritic":               "ugar",
	"Buginese":               "bugi",
	"Coptic":                 "copt",
	"Glagolitic":             "glag",
	"Kharoshthi":             "khar",
	"New_Tai_Lue":            "talu",
	"Old_Persian":            "xpeo",
	"Syloti_Nagri":           "sylo",
	"Tifinagh":               "tfng",
	"Balinese":               "bali",
	"Cuneiform":              "xsux",
	"Nko":                    "nkoo",
	"Phags_Pa":               "phag",
	"Phoenician":             "phnx",
	"Carian":                 "cari",
	"Cham":                   "cham",
	"Kayah_Li":               "kali",
	"Lepcha":                 "lepc",
	"Lycian":                 "lyci",
	"Lydian":                 "lydi",
	"Ol_Chiki":               "olck",
	"Rejang":                 "rjng",
	"Saurashtra":             "saur",
	"Sundanese":              "sund",
	"Vai":                    "vaii",
	"Avestan":                "avst",
	"Bamum":                  "bamu",
	"Egyptian_Hieroglyphs":   "egyp",
	"Imperial_Aramaic":       "armi",
	"Inscriptional_Pahlavi":  "phli",
	"Inscriptional_Parthian": "prti",
	"Javanese":               "java",
	"Kaithi":                 "kthi",
	"Lisu":                   "lisu",
	"Meetei_Mayek":           "mtei",
	"Old_South_Arabian":      "sarb",
	"Old_Turkic":             "orkh",
	"Samaritan":              "samr",
	"Tai_Tham":               "lana",
	"Tai_Viet":               "tavt",
	"Batak":                  "batk",
	"Brahmi":                 "brah",
	"Mandaic":                "mand",
	"Chakma":                 "cakm",
	"Meroitic_Cursive":       "merc",
	"Meroitic_Hieroglyphs":   "mero",
	"Miao":                   "plrd",
	"Sharada":                "shrd",
	"Sora_Sompeng":           "sora",
	"Takri":                  "takr",
	"Bassa_Vah":              "bass",
	"Caucasian_Albanian":     "aghb",
	"Duployan":               "dupl",
	"Elbasan":                "elba",
	"Grantha":                "gran",
	"Khojki":                 "khoj",
	"Khudawadi":              "sind",
	"Linear_A":               "lina",
	"Mahajani":               "mahj",
	"Manichaean":             "mani",
	"Mende_Kikakui":          "mend",
	"Modi":                   "modi",
	"Mro":                    "mroo",
	"Nabataean":              "nbat",
	"Old_North_Arabian":      "narb",
	"Old_Permic":             "perm",
	"Pahawh_Hmong":           "hmng",
	"Palmyrene":              "palm",
	"Pau_Cin_Hau":            "pauc",
	"Psalter_Pahlavi":        "phlp",
	"Siddham":                "sidd",
	"Tirhuta":                "tirh",
	"Warang_Citi":            "wara",
	"Ahom":                   "ahom",
	"Anatolian_Hieroglyphs":  "hluw",
	"Hatran":                 "hatr",
	"Multani":                "mult",
	"Old_Hungarian":          "hung",
	"Signwriting":            "sgnw",
	"Adlam":                  "adlm",
	"Bhaiksuki":              "bhks",
	"Marchen":                "marc",
	"Osage":                  "osge",
	"Tangut":                 "tang",
	"Newa":                   "newa",
	"Masaram_Gondi":          "gonm",
	"Nushu":                  "nshu",
	"Soyombo":                "soyo",
	"Zanabazar_Square":       "zanb",
	"Dogra":                  "dogr",
	"Gunjala_Gondi":          "gong",
	"Hanifi_Rohingya":        "rohg",
	"Makasar":                "maka",
	"Medefaidrin":            "medf",
	"Old_Sogdian":            "sogo",
	"Sogdian":                "sogd",
	"Elymaic":                "elym",
	"Nandinagari":            "nand",
	"Nyiakeng_Puachue_Hmong": "hmnp",
	"Wancho":                 "wcho",
	"Chorasmian":             "chrs",
	"Dives_Akuru":            "diak",
	"Khitan_Small_Script":    "kits",
	"Yezidi":                 "yezi",
}

// We do support this list of languages.
var supportedLanguages = map[language.Tag]string{
	language.Arabic:     "ARA",
	language.Chinese:    "ZHS",
	language.English:    "ENG",
	language.Greek:      "ELL",
	language.German:     "DEU",
	language.Hebrew:     "IWR",
	language.Japanese:   "JAN",
	language.Portuguese: "PTG",
	language.Romanian:   "ROM",
	language.Russian:    "RUS",
	language.Turkish:    "TRK",
}

// We will try to match user-preferred language against supported languages.
var supportedLanguagesMatcher language.Matcher

func init() {
	// prepare the language matcher with our list of supported languages
	langs := make([]language.Tag, len(supportedLanguages))
	i := 0
	for l := range supportedLanguages {
		langs[i] = l
		i++
	}
	supportedLanguagesMatcher = language.NewMatcher(langs)
}

// ScriptTagForScript returns the appropriate OpenType script tag for a given ISO 15924
// script code. It will return the DFLT-tag for unknown or unsupported scripts.
func ScriptTagForScript(script language.Script) ot.Tag {
	s := script.String()
	if otScr, ok := script2opentype[s]; ok {
		return ot.T(otScr)
	}
	return ot.DFLT
}

// LanguageTagForLanguage returns the appropriate OpenType language tag for a given
// BCP 47 language tag.
// If there is no supported language, that can be matched with confidence of at least `conf`,
// the DFLT-tag will be returned.
func LanguageTagForLanguage(lang language.Tag, conf language.Confidence) ot.Tag {
	l, _, c := supportedLanguagesMatcher.Match(lang)
	tracer().Debugf("OpenType language matched %s (%s) : %s", display.English.Tags().Name(l),
		display.Self.Name(l), c)
	if c < conf { // if matcher's confidence level is not high enough
		return ot.DFLT
	}
	base, _ := language.Compose(l.Base()) // re-package l to cleanly match base language constant
	if ltag, ok := supportedLanguages[base]; ok {
		return ot.T(ltag)
	}
	return ot.DFLT
}

// For some script/language combinations the Unicde de-composed (NFD) is the preferred
// form for later states of the shaping pipeline.
// If the language list contains just DFLT, the script prefers NFD independent of the language.
var scriptPreferDecomposed = map[ot.Tag][]ot.Tag{
	ot.T("dev2"): {ot.DFLT}, // all Devangari flavours
	ot.T("bng2"): {ot.DFLT}, // all Bengali flavours
}

// prefersDecomposed signals wether a script should be de-composed before shaping.
// For some script/language combinations the Unicde de-composed (NFD) is the preferred
// form for later states of the shaping pipeline.
func prefersDecomposed(script ot.Tag, lang ot.Tag) bool {
	if langs, ok := scriptPreferDecomposed[script]; ok {
		if len(langs) > 0 {
			if langs[0] == ot.DFLT || slices.Contains(langs, lang) {
				return true
			}
		}
	}
	return false
}
