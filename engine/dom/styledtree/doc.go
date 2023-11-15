/*
Package styledtree is a straightforward default implementation of a styled document tree.

Overview

This is an implementation of style.TreeNode and of cssom.StyledNode.
Using a builder type, cssom.Style() will create a styled tree from an
HTML parse tree and a CSSOM.

___________________________________________________________________________

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>

*/
package styledtree

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'tyse.dom'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.dom")
}
