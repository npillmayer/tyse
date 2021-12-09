/*
License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package parameters

import (
	"golang.org/x/text/unicode/bidi"

	"github.com/npillmayer/tyse/core/dimen"
)

type TypesettingParameter int

//go:generate stringer -type=TypesettingParameter
const (
	none TypesettingParameter = iota
	P_LANGUAGE
	P_SCRIPT
	P_TEXTDIRECTION
	P_BASELINESKIP
	P_LINESKIP
	P_LINESKIPLIMIT
	P_HYPHENCHAR
	P_HYPHENPENALTY
	P_MINHYPHENLENGTH
	P_STOPPER
)

type ParameterGroup struct {
	params map[TypesettingParameter]interface{}
	level  int
	next   *ParameterGroup
}

type TypesettingRegisters struct {
	base       [P_STOPPER]interface{}
	groups     *ParameterGroup
	grouplevel int
}

// ----------------------------------------------------------------------

func NewTypesettingRegisters() *TypesettingRegisters {
	regs := &TypesettingRegisters{}
	initParameters(&regs.base)
	return regs
}

func initParameters(p *[P_STOPPER]interface{}) {
	p[P_LANGUAGE] = "en_EN"               // a string
	p[P_SCRIPT] = "Latin"                 // a string
	p[P_TEXTDIRECTION] = bidi.LeftToRight //
	p[P_BASELINESKIP] = 12 * dimen.PT     // dimension
	p[P_LINESKIP] = 0                     // dimension
	p[P_LINESKIPLIMIT] = 0                // dimension
	p[P_HYPHENCHAR] = int('-')            // a rune
	p[P_HYPHENPENALTY] = 0                // a numeric penalty (int)
	p[P_MINHYPHENLENGTH] = dimen.Infinity // a numeric quantitiv (int) = # of runes
}

func (regs *TypesettingRegisters) Begingroup() {
	regs.grouplevel++
}

func (regs *TypesettingRegisters) Endgroup() {
	if regs.grouplevel > 0 {
		if regs.groups != nil && regs.groups.level == regs.grouplevel {
			regs.groups = regs.groups.next
			regs.grouplevel--
		}
	}
}

func (regs *TypesettingRegisters) Push(key TypesettingParameter, value interface{}) {
	if regs.grouplevel > 0 {
		var g *ParameterGroup
		if regs.groups == nil {
			g = &ParameterGroup{}
			g.params = make(map[TypesettingParameter]interface{})
			g.level = regs.grouplevel
			regs.groups = g
		} else {
			if regs.groups.level < regs.grouplevel {
				g = &ParameterGroup{}
				g.params = make(map[TypesettingParameter]interface{})
				g.level = regs.grouplevel
				g.next = regs.groups
				regs.groups = g
			} else {
				g = regs.groups
			}
		}
		g.params[key] = value
	} else {
		regs.base[key] = value
	}
}

func (regs *TypesettingRegisters) Get(key TypesettingParameter) interface{} {
	if key <= 0 || key == P_STOPPER {
		panic("parameter key outside range of typesetting parameters")
	}
	var value interface{}
	if regs.grouplevel > 0 {
		for g := regs.groups; g != nil; g = g.next {
			value = g.params[key]
			if value != nil {
				break
			}
		}
	}
	if value == nil {
		value = regs.base[key]
	}
	return value
}

func (regs *TypesettingRegisters) S(key TypesettingParameter) string {
	return regs.Get(key).(string)
}

func (regs *TypesettingRegisters) N(key TypesettingParameter) int {
	return regs.Get(key).(int)
}

func (regs *TypesettingRegisters) D(key TypesettingParameter) dimen.DU {
	return regs.Get(key).(dimen.DU)
}
