package layout

import (
	"errors"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/dom/style/css"
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
  different approaches to layout. It's easier to reason about layout in a recursive
  top-down manner (as does Mozilla), but I intend to move to a more concurrent
  approach step-by-step.
*/

var ErrHeightNotFixed error = errors.New("height of context not determined")
var ErrEnclosingWidthNotFixed error = errors.New("enclosing width not fixed")
var ErrNotAPercentageDimension error = errors.New("input dimension not a percentage dimension")

// var ErrUnfixedscaledUnit error = errors.New("font/view dependent dimension must be fixed")
// var ErrContentScaling error = errors.New("box scales with content")

/*
  For top-down measurements, enclosing containers are responsible for calculation the
  enclosing width for sub-containers (constraints). This includes checking the type of the child
  as well as checking for child.IsFlowRoot() and possibly subtracting float widths.
*/

/*
  We normalize parameters for two reasons:
  1. parameters tend to be numerous for layout functions and mostly are very similar
  2. I intend to parallelize tree traversals for layout in the future, and for this
     we need a clear understanding of input vs output.
*/
type inheritedParams struct {
	flowRoot   *frame.FlowRoot
	view       *View
	W          css.DimenT // enclosing width
	MinW, MaxW dimen.DU   // constraints
}

type synthesizedParams struct {
	W       dimen.DU
	H       dimen.DU
	lastErr error
}

type View struct {
	Width dimen.DU
}

func BoxTreeToLayoutTree(boxRoot *boxtree.PrincipalBox, view *View) (syn synthesizedParams) {
	tracer().Debugf("================= ###### Layout ###### =====================")
	if view == nil {
		if boxRoot != nil {
			syn.lastErr = errors.New("illegal arguments: view is void")
		}
		return
	}
	params := inheritedParams{
		W:    css.SomeDimen(view.Width),
		MinW: 0,
		MaxW: view.Width,
		view: view,
	}
	if !boxRoot.Display.Outer().Contains(css.BlockMode) {
		syn.lastErr = errors.New("layout root expected to have display mode block")
	}
	boxRoot.SetContext(NewContextFor(&boxRoot.ContainerBase))
	// This should be deferred to CalcBlockWidths, even for #document ?!
	if !boxRoot.Context.IsFlowRoot() {
		syn.lastErr = errors.New("layout root expected to be flow root")
	} else {
		params.flowRoot = boxRoot.Context.FlowRoot()
		syn = CalcBlockWidths(&boxRoot.ContainerBase, params)
	}
	if syn.lastErr != nil {
		tracer().Errorf("layout tree error: %v", syn.lastErr)
	}
	tracer().Debugf("=================== ############### ======================")
	// TODO
	// now recursivly call boxRoot.Context().Layout(params/flowRoot)
	return syn
}

// Potentially recursive call to nested containers
func CalcBlockWidths(c *frame.ContainerBase, inherited inheritedParams) (syn synthesizedParams) {
	if c.Context == nil {
		c.Context = NewContextFor(c)
	}
	tracer().Debugf("calculating block width of [%s]", boxtree.ContainerName(c))
	// case c.Box.W is Font or View dependent: should have been done already => error
	// case c.Box.W is Content dependent: call calc on nested block
	// case c.Box.W is absolute: we're done
	/*if w, err := c.CSSBox().TotalWidth().Match(option.Of{
		option.None:       nil, // defaults to `auto`
		css.FontScaled:    option.Fail(ErrUnfixedscaledUnit),
		css.ViewScaled:    option.Fail(ErrUnfixedscaledUnit),
		css.FixedValue:    c.CSSBox().TotalWidth().Unwrap(),
		css.ContentScaled: nil,
		option.Some:       nil,
	}); w != nil {
		syn.W = w.(dimen.Dimen)
		return
	} else if err != nil {
		return withError(syn, err)
	}*/
	if w := c.CSSBox().TotalWidth(); w.IsAbsolute() {
		syn.W = w.Unwrap()
		return
	}
	// if c.ctx.isFlowRoot:
	//      tell flow root to layout floats
	//      wait for shelf-line condition
	//      subtract floats' width from enclosing width
	//
	// Now we're ready to:
	//syn = solveWidthTopDown(c, inherited)
	var ok bool
	if inherited.W.IsAbsolute() {
		ok, syn.lastErr = frame.FixDimensionsFromEnclosingWidth(c.CSSBox(), inherited.W.Unwrap())
	} else {
		syn.lastErr = frame.ErrContentScaling
	}
	// recursion step
	if ok && syn.lastErr == nil {
		// recurse down
		if hasContained := c.RenderNode().PresetContained(); hasContained {
			tracer().Debugf("calculating width for %d children of [%s]", len(c.Context.Contained()),
				boxtree.ContainerName(c))
			inherited.W = c.CSSBox().W
			inherited.MaxW = c.CSSBox().W.Unwrap()
			for _, sub := range c.Context.Contained() {
				//if sub.Type() == boxtree.TypeText {
				if boxtree.IsText(sub.RenderNode()) {
					continue
				}
				tracer().Debugf("--------------------------------------------------")
				tracer().Debugf("width of sub-container [%s]", boxtree.ContainerName(sub))
				s := CalcBlockWidths(sub, inherited)
				if s.lastErr != nil {
					syn.lastErr = s.lastErr
					return
				}
			}
		}
		// now c should have its width determined
		// and the non-text sub-containers of c should have their dimensions fixed
		tracer().Debugf("calling [%s].Layout()", boxtree.ContainerName(c))
		syn.lastErr = c.Context.Layout(inherited.flowRoot)
	} else if syn.lastErr == frame.ErrContentScaling {
		tracer().Debugf("[%v] width cannot be solved top-down, have to calc content-based",
			c.DOMNode().NodeName())
		syn = solveWidthForContent(c, inherited) // this will recurse all the way down
	}
	if syn.lastErr != nil {
		tracer().Debugf("calc block width: last error = %v", syn.lastErr)
	}
	if size, _, _ := c.Context.Measure(); !size.H.IsNone() {
		syn.H = size.H.Unwrap()
	} else {
		return withError(syn, ErrHeightNotFixed)
	}
	return
}

func LayoutBlockFormattingContext(ctx frame.Context, flowRoot *frame.FlowRoot) *frame.Box {
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
	// layout B:
	//     find width of B and fix margins
	//     layout B.ctx
	// collect container B
	//     append B with relative offset to C
	//
	return nil
}

func LayoutInlineFormattingContext(ctx frame.Context, flowRoot *frame.FlowRoot) *frame.Box {
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

// enclosing is width of containing block.
// will stay inside c, no traversing of children containers.
// Instead will flag an appropriate error, which the caller will use to traverse
// nested containers before calling Solve… again.
//
// Will distribute space according to the equation (ref. CSS spec):
//
//     margin-left + width + margin-right = width of containing block
//
/*
func solveWidthTopDown(c frame.Container, inherited inheritedParams) (syn synthesizedParams) {
	// TODO fix relative width,margins => resolve against enclosing
	// => should already been done
	width := c.CSSBox().ContentWidth()
	calc, err := width.Match(option.Of{
		option.None:       calcWidthAsRest, // default is `auto`
		css.Auto:          calcWidthAsRest,
		css.ContentScaled: option.Fail(ErrContentScaling),
		option.Some:       takeWidth,
	})
	if err != nil {
		return withError(syn, err)
	}
	T().Debugf("solving width of [%v] top down, width now = %v", boxtree.ContainerName(c), width)
	solve := asCalcFn(calc)
	w, _ := solve(c, width, inherited.W)
	box := distributeMarginSpace(c, w.Unwrap(), inherited.W.Unwrap())
	// TODO how to proceed with box?
	// -> assign to CSS box of c
	T().Debugf("solved width of [%v] to %s", boxtree.ContainerName(c), box.TotalWidth())
	if !box.W.IsAbsolute() {
		return withError(syn, "box width not determined after margin space distribution")
	}
	syn.W = box.TotalWidth().Unwrap()
	return
}
*/

func solveWidthForContent(c *frame.ContainerBase, inherited inheritedParams) (syn synthesizedParams) {
	panic("TODO")
}

func SolveWidthBottomUp(c *frame.ContainerBase, enclosing dimen.DU) (*frame.Box, error) {
	panic("TODO")
}

// --- Various dimen constraint solving strategies ---------------------------
/*
type calcFn func(c frame.Container, w, enclosing css.DimenT) (css.DimenT, error)

func asCalcFn(f interface{}) calcFn {
	return f.(func(c frame.Container, w, enclosing css.DimenT) (css.DimenT, error))
}

func takeWidth(c frame.Container, w, enclosing css.DimenT) (css.DimenT, error) {
	T().Debugf("calculating width: simply take is as is = %v", w)
	return c.CSSBox().ContentWidth(), nil
}

// Spec: If 'width' is set to 'auto', any other 'auto' values become '0'
// and 'width' follows from the resulting equality.
func calcWidthAsRest(c frame.Container, w, enclosing css.DimenT) (css.DimenT, error) {
	left := fixDimensionMust(c.CSSBox().Margins[frame.Left], c, enclosing)
	c.CSSBox().Margins[frame.Left] = css.SomeDimen(left) // do not lose fixed value
	right := fixDimensionMust(c.CSSBox().Margins[frame.Right], c, enclosing)
	c.CSSBox().Margins[frame.Right] = css.SomeDimen(right) // do not lose fixed value
	width := enclosing.Unwrap() - left - right
	r := css.SomeDimen(width)
	T().Debugf("calculate width as rest (without decoration) to %s", r)
	return r, nil
	//return css.SomeDimen(width), nil
}
*/
// ---------------------------------------------------------------------------

// This must not be called for dimensions in unfixed relative units, except '%'.
/*
func fixDimensionMust(d css.DimenT, c frame.Container, enclosing css.DimenT) (fixedDimen dimen.Dimen) {
	var err error
	var fixed interface{}
	if d.IsRelative() {
		fixedDimen, err = fixRelativeDimension(d, c, enclosing)
	} else {
		fixed, err = d.Match(option.Of{ // TODO function MatchToDimen ? frequent case ?
			option.None: dimen.Zero,
			css.Initial: dimen.Zero,
			css.Auto:    dimen.Zero,
			option.Some: d.Unwrap(), // TODO safety: this will give nonsense for unfixed relative units
		})
		if fixed != nil {
			fixedDimen = fixed.(dimen.Dimen)
		}
	}
	if err != nil {
		T().Errorf("layout fix relative dimen: %s", err.Error())
		return dimen.Zero
	}
	return fixedDimen
}

// Forbidden to have side effects !
func fixRelativeDimension(d css.DimenT, c frame.Container, w css.DimenT) (dimen.Dimen, error) {
	if d.IsAbsolute() {
		return d.Unwrap(), nil
	}
	enclosing := w.Unwrap()
	width, err := d.Match(option.Of{
		option.None: dimen.Zero,
		"%":         d.Unwrap() * enclosing / 100,
		option.Some: option.Fail(ErrNotAPercentageDimension),
		// These 2 should both be done during boxtree buildup:
		// TODO css.FontScaled: d.ScaleFromFont(c.DOMNode().Font()),
		// TODO css.ViewScaled: d.ScaleFromView(...)
	})
	if err != nil {
		T().Errorf("layout fix %%-dimen: %s", err.Error())
		return dimen.Zero, err
	}
	return width.(dimen.Dimen), nil
}

// w and enclosing should be fixed
func distributeMarginSpace(c frame.Container, w, enclosing dimen.Dimen) *frame.Box {
	//box := &frame.Box{}
	box := c.CSSBox() // make a copy of c's CSS box
	remaining := enclosing - w
	if remaining == 0 { // TODO fit into general case
		box.FixBorderBoxWidth(w)
	} else {
		left := c.CSSBox().Margins[frame.Left]
		right := c.CSSBox().Margins[frame.Right]
		l, _ := left.Match(option.Of{
			css.Auto: option.Safe(right.Match(option.Of{
				css.Auto:    remaining / 2,
				option.Some: remaining - right.Unwrap(),
			})),
		})
		r := remaining - l.(dimen.Dimen)
		box.Margins[frame.Left] = css.SomeDimen(l.(dimen.Dimen))
		box.Margins[frame.Right] = css.SomeDimen(r)
		box.FixBorderBoxWidth(w)
	}
	return box
}

func setWidthFromParent(c frame.Container, enclosing dimen.Dimen) bool {
	dw := c.CSSBox().DecorationWidth(true)
	if !dw.IsAbsolute() {
		return false
	}
	return c.CSSBox().FixContentWidth(dw.Unwrap())
}
*/

func withError(syn synthesizedParams, arg interface{}) synthesizedParams {
	switch a := arg.(type) {
	case string:
		syn.lastErr = errors.New(a)
	case error:
		syn.lastErr = a
	default:
		syn.lastErr = errors.New("error")
	}
	return syn
}
