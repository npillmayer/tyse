package ot

import "github.com/npillmayer/schuko/tracing"

func GposDebugInfo(otf *OTFont) {
	level := trace().GetTraceLevel()
	trace().SetTraceLevel(tracing.LevelInfo)
	defer trace().SetTraceLevel(level)
	_, err := otf.ot.GposTable()
	if err != nil {
		trace().Errorf("cannot read GPOS table of OpenType font %s", otf.f.Fontname)
		trace().Errorf(err.Error())
		return
	}
	trace().Infof("OpenType GPOS table of %s", otf.f.Fontname)
}
