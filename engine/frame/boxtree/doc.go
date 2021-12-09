/*
Package boxtree produces a box-tree from a styled tree (DOM).

______________________________________________________________________

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package boxtree

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'tyse.frame.box'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.frame.box")
}
