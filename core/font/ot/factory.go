package ot

import "fmt"

type Navigator interface {
	Name() string
	Link() Link
	Map() TagRecordMap
	List() List
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
	case "name":
		names, err := parseNames(loc.Bytes())
		if err != nil {
			return null(err)
		}
		return names
	case "NameRecord":
		name, err := decodeUtf16(loc.Bytes())
		if err != nil {
			return null(err)
		}
		return navName{name: name}
	}
	if fields, ok := tableFields[obj]; ok {
		trace().Debugf("object %s has fields %v", obj, fields)
		f := otFields{pattern: fields, b: base}
		trace().Debugf("b[0..10] = %v", base[:10])
		return list{navName: navName{name: obj}, f: f}
	}
	trace().Debugf("no navigator found -> null navigator")
	return null(errDanglingLink(obj))
}

var tableFields = map[string][]uint8{
	"OS/2": {2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 10, 4, 1, 2, 2, 2},
}

type navBase struct {
	err error
}

func (nbase navBase) Link() Link {
	return nullLink("generic link class")
}

func (nbase navBase) Map() TagRecordMap {
	return tagRecordMap16{}
}

func (nbase navBase) List() List {
	return nullList
}

func (nbase navBase) IsVoid() bool {
	return true
}

func (nbase navBase) Name() string {
	if nbase.err != nil {
		return nbase.err.Error()
	}
	return "base nav: should not be visible"
}

func (nbase navBase) Error() error {
	return nbase.err
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

func (lm linkAndMap) List() List {
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

func null(err error) Navigator {
	return navBase{err: err}
}

func nullLink(errmsg string) Link {
	return link16{err: fmt.Errorf("link: %s", errmsg)}
}

func errDanglingLink(obj string) error {
	return fmt.Errorf("cannot resolve link to %s", obj)
}

var nullNav = linkAndMap{}
var nullList = u16List{}

type navName struct {
	navBase
	name string
}

func (nm navName) Name() string {
	return nm.name
}

type list struct {
	navName
	f otFields
}

func (l list) List() List {
	return l.f
}

type otFields struct {
	pattern []uint8
	b       fontBinSegm
}

func (f otFields) Len() int {
	return len(f.pattern)
}

func (f otFields) Get(i int) []byte {
	if i < 0 || i >= len(f.pattern) {
		return []byte{}
	}
	offset := 0
	for j, p := range f.pattern {
		if j > i {
			break
		}
		offset += int(p)
	}
	if r, err := f.b.view(offset, int(f.pattern[i])); err == nil {
		return r
	}
	return []byte{}
}

func (f otFields) All() [][]byte {
	r := make([][]byte, len(f.pattern))
	offset := 0
	for _, p := range f.pattern {
		if x, err := f.b.view(offset, int(p)); err == nil {
			r = append(r, x)
		}
		return [][]byte{}
	}
	return r
}
