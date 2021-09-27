package ot

var DFLT = T("DFLT")

var standardScripts = []Tag{
	T("latn"), // Latin
	T("cyrl"), // Cyrillic
	T("grek"), // Greek
	T("armn"), // Armenian
	T("geor"), // Georgian
	T("runr"), // Runic
	T("ogam"), // Ogham
}

var complexScripts = []Tag{
	T("adlm"), // ADLaM
	T("ahom"), // Ahom
	T("bhks"), // Bhaiksuki
	T("bali"), // Balinese
	T("batk"), // Batak
	T("brah"), // Brahmi
	T("bugi"), // Buginese
	T("buhd"), // Buhid
	T("cakm"), // Chakma
	T("cham"), // Cham
	T("chrs"), // Chorasmian
	T("diak"), // Dives Akuru
	T("dogr"), // Dogra
	T("dupl"), // Duployan
	T("elym"), // Elymaic
	T("gran"), // Grantha
	T("gong"), // Gunjala Gondi
	T("rohg"), // Hanifi Rohingya
	T("hano"), // Hanunoo
	T("java"), // Javanese
	T("kthi"), // Kaithi
	T("kali"), // Kayah Li
	T("khar"), // Kharoshthi
	T("kits"), // Khitan Small Script
	T("khoj"), // Khojki
	T("sind"), // Khudawadi
	T("lepc"), // Lepcha
	T("limb"), // Limbu
	T("mahj"), // Mahajani
	T("maka"), // Makasar
	T("mand"), // Mandaic
	T("mani"), // Manichaean
	T("marc"), // Marchen
	T("gonm"), // Masaram Gondi
	T("medf"), // Medefaidrin
	T("mtei"), // Meitei Mayek
	T("plrd"), // Miao
	T("modi"), // Modi
	T("mong"), // Mongolian
	T("mult"), // Multani
	T("nand"), // Nandinagari
	T("newa"), // Newa
	T("hmnp"), // Nyiakeng_Puachue_Hmong
	T("sogo"), // Old_Sogdian
	T("hmng"), // Pahawh Hmong
	T("phag"), // Phags-pa
	T("phlp"), // Psalter Pahlavi
	T("rjng"), // Rejang
	T("saur"), // Saurashtra
	T("shrd"), // Sharada
	T("sidd"), // Siddham
	T("sinh"), // Sinhala
	T("sogd"), // Sogdian
	T("soyo"), // Soyombo
	T("sund"), // Sundanese
	T("sylo"), // Syloti Nagri
	T("tglg"), // Tagalog
	T("tagb"), // Tagbanwa
	T("tale"), // Tai_Le
	T("lana"), // Tai_Tham
	T("tavt"), // Tai_Viet
	T("takr"), // Takri
	T("tibt"), // Tibetan
	T("tfng"), // Tifinagh
	T("tirh"), // Tirhuta
	T("wcho"), // Wancho
	T("yezi"), // Yezidi
	T("zanb"), // Zanabazar Square
}

var semiticScripts = []Tag{
	T("arab"), // Arabic
	T("hebr"), // Hebrew
}

var indicScripts = []Tag{
	T("bng2"), // Bengali
	T("dev2"), // Devanagari
}
