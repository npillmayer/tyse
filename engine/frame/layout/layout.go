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

/*
  It would be natural to make the layout functions members of type Context.
  However, I prefer setting up a driver pattern to be able to experiment with
  different approaches to layout. I's easier to reason about layout in a recursive
  top-down manner (as does Mozilla), but I intend to move to a more concurrent
  approach step-by-step.
*/

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

func LayoutInlineFormattingContext(ctx boxtree.Context, flowRoot *boxtree.FlowRoot) *frame.Box {
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
	// encode text of C
	//   - packging block boxes into replacable sub-boxes S with parbreak penalties
	//   - package inline-block boxes into sub-boxis S with inline penalties
	// set S.w to w if S.display = block
	// layout S.ctx
	//
	return nil
}
