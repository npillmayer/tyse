/*
Package layout produces a render tree from a styled tree.

Overview

Early draft, nothing useful here yet.

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package layout

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'tyse.frame'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.frame")
}
