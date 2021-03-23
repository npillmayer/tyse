package ot

import "fmt"

type Navigator interface {
	Name() string
	Link() Link
	Map() TagRecordMap
	List() []uint16
	IsVoid() bool
	Error() error
}

func navFactory(obj string, loc Location, base fontBinSegm) Navigator {
	trace().Debugf("navigator factory for %s", obj)
	switch obj {
	case "Script":
		l, err := parseLink16(loc.Bytes(), 0, loc.Bytes(), "LangSys")
		if err != nil {
			return null(err)
		}
		return linkAndMap{
			link: l,
			tmap: parseTagRecordMap16(loc.Bytes(), 2, loc.Bytes(), "Script", "LangSys"),
		}
	case "LangSys":
		trace().Debugf("%s[0] = %x", obj, u16(loc.Bytes()))
		lsys, err := parseLangSys(loc.Bytes(), 2, "int")
		if err != nil {
			trace().Errorf(err.Error()) // TODO carry in navigator chain
		}
		return lsys
	}
	trace().Debugf("no navigator found -> null navigator")
	return null(errDanglingLink("obj"))
}

type linkAndMap struct {
	err  error
	link Link
	tmap TagRecordMap
}

func (lm linkAndMap) Link() Link {
	return lm.link
}

func (lm linkAndMap) Map() TagRecordMap {
	return lm.tmap
}

func (lm linkAndMap) List() []uint16 {
	return nullList
}

func (lm linkAndMap) IsVoid() bool {
	return lm.tmap.Count() == 0
}

func (lm linkAndMap) Name() string {
	return lm.tmap.Name()
}

func (lm linkAndMap) Error() error {
	return lm.err
}

func null(err error) linkAndMap {
	return linkAndMap{err: err}
}

func nullLink(errmsg string) Link {
	return link16{err: fmt.Errorf("link: %s", errmsg)}
}

func errDanglingLink(obj string) error {
	return fmt.Errorf("cannot resolve link to %s", obj)
}

var nullNav = linkAndMap{}
var nullList = []uint16{}
