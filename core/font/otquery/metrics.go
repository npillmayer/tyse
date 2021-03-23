package otquery

import (
	"strings"

	"github.com/npillmayer/tyse/core/font/ot"
	"golang.org/x/text/language"
)

func SupportsScript(otf *ot.Font, scr language.Script) (string, string) {
	t := otf.Table(ot.T("GSUB"))
	if t == nil {
		// do nothing
	}
	gsub := t.Base().AsGSub()
	scrTag := ot.T(strings.ToLower(scr.String()))
	rec := gsub.Scripts.Lookup(scrTag)
	if rec.IsNull() {
		trace().Debugf("cannot find script %s in font", scr.String())
	} else {
		trace().Debugf("script %s is contained in GSUB", scr.String())
		s := rec.Navigate()
		for _, tag := range s.Map().Tags() {
			trace().Debugf("tag = %s", tag.String())
			l := s.Map().Lookup(tag)
			lsys := l.Navigate()
			trace().Debugf("list = %v", lsys.List())
		}
	}
	return "DFLT", "DFLT"
}
