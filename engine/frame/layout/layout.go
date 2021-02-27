package layout

import (
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
)

// Invaluable:
// https://developer.mozilla.org/en-US/docs/Web/CSS/Visual_formatting_model
//
// Regions:
// http://cna.mamk.fi/Public/FJAK/MOAC_MTA_HTML5_App_Dev/c06.pdf

func LayoutBlockFormattingContext(ctx boxtree.Context, flowRoot *boxtree.FlowRoot) *frame.Box {
	// C = ctx.container
	// C.w is already set
	if ctx.IsFlowRoot() {
		// ignore flowRoot.Floats
		// collect floats from C.children
		// flowRoot = ctx
	}
	//
	// layout floats
	//
	// alternative T:
	// collect contiguous text nodes from C
	// pack them in an anonymous box ANON with inline context
	// set ANON.w to C.w
	// layout ANON.ctx
	// (ANON is at end of ctx)
	//
	// alternative B:
	// collect container B
	// layout B.ctx
	//
	return nil
}
