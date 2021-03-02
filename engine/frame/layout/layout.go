package layout

import (
	"errors"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/option"
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
  different approaches to layout. I's easier to reason about layout in a recursive
  top-down manner (as does Mozilla), but I intend to move to a more concurrent
  approach step-by-step.
*/

var ErrUnfixedscaledUnit error = errors.New("font/view dependent dimension must be fixed")
var ErrEnclosingWidthNotFixed error = errors.New("enclosing width not fixed")
var ErrContentScaling error = errors.New("box scales with content")
var ErrNotAPercentageDimension error = errors.New("input dimension not a percentage dimension")

/*
  For top-down measurements, enclosing containers are responsible for calculation
  the enclosing width for sub-containers. This includes checking the type of the child
  as well as checking for child.IsFlowRoot() and possibly subtracting float widths.
*/

/*
  We normalize parameters for two reasons:
  1. parameters tend to be numerous for layout functions and mostly are very similar
  2. I intend to parallelize tree traversals for layout in the future, and for this
     we need a clear understanding of input vs output.
*/
type inheritedParams struct {
	flowRoot *boxtree.FlowRoot
	W        css.DimenT // enclosing width    fixed ?
}

type synthesizedParams struct {
	W       dimen.Dimen
	H       dimen.Dimen
	lastErr error
}

// potentially recursive call to nested containers
func CalcBlockWidth(c boxtree.Container, inherited inheritedParams) (syn synthesizedParams) {
	// case c.Box.W is Font or View dependent: should have been done already => error
	// case c.Box.W is Content dependent: call calc on nested block
	// case c.Box.W is absolute: we're done
	w, err := c.CSSBox().TotalWidth().Match(option.Of{
		option.None:    nil, // defaults to `auto`
		css.FontScaled: option.Fail(ErrUnfixedscaledUnit),
		css.ViewScaled: option.Fail(ErrUnfixedscaledUnit),
		css.FixedValue: c.CSSBox().TotalWidth().Unwrap(),
		option.Some:    nil,
	})
	if err != nil {
		syn.lastErr = err
		return
	}
	if w != nil {
		syn.W = w.(dimen.Dimen)
		return
	}
	// if c.ctx.isFlowRoot:
	//      tell flow root to layout floats
	//      subtract floats' width from enclosing width
	//
	// Now we're ready to:
	// SolveWidthTopDown(c boxtree.Container, enclosing dimen.Dimen) (*frame.Box, error)
	//
	return
}

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
	// layout B:
	//     find width of B and fix margins
	//     layout B.ctx
	// collect container B
	//     append B with relative offset to C
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

// enclosing is width of containing block.
// will stay inside c, not traversing of children containers.
// Instead will flag an appropriate error, which the caller will use to traverse
// nested containers before calling Solve… again.
//
// Will distribute space according to the equation (ref. CSS spec):
//
//     margin-left + width + margin-right = width of containing block
//
func SolveWidthTopDown(c boxtree.Container, enclosing dimen.Dimen) (*frame.Box, error) {
	var box *frame.Box
	// TODO fix relative width,margins => resolve against enclosing
	width := c.CSSBox().Width()
	calc, err := width.Match(option.Of{
		option.None:       calcWidthAsRest, // default is `auto`
		css.Auto:          calcWidthAsRest,
		css.ContentScaled: option.Fail(ErrContentScaling),
		option.Some:       takeWidth,
	})
	if err != nil {
		return c.CSSBox(), err
	}
	solve := calc.(calcFn)
	w, _ := solve(c, width, css.SomeDimen(enclosing))
	box = distributeMarginSpace(c, w.Unwrap(), enclosing)
	return box, nil
}

func SolveWidthBottomUp(c boxtree.Container, enclosing dimen.Dimen) (*frame.Box, error) {
	panic("TODO")
}

// --- Various dimen constraint solving strategies ---------------------------

type calcFn func(c boxtree.Container, w, enclosing css.DimenT) (css.DimenT, error)

func takeWidth(c boxtree.Container, w, enclosing css.DimenT) (css.DimenT, error) {
	return w, nil
}

// Spec: If 'width' is set to 'auto', any other 'auto' values become '0'
// and 'width' follows from the resulting equality.
func calcWidthAsRest(c boxtree.Container, w, enclosing css.DimenT) (css.DimenT, error) {
	left := fixDimensionMust(c.CSSBox().Margins[frame.Left], c, enclosing)
	c.CSSBox().Margins[frame.Left] = css.SomeDimen(left) // do not lose fixed value
	right := fixDimensionMust(c.CSSBox().Margins[frame.Right], c, enclosing)
	c.CSSBox().Margins[frame.Right] = css.SomeDimen(right) // do not lose fixed value
	width := enclosing.Unwrap() - left - right
	return css.SomeDimen(width), nil
}

// ---------------------------------------------------------------------------

// This must not be called for dimensions in unfixed relative units, except '%'.
func fixDimensionMust(d css.DimenT, c boxtree.Container, enclosing css.DimenT) (fixedDimen dimen.Dimen) {
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
func fixRelativeDimension(d css.DimenT, c boxtree.Container, w css.DimenT) (dimen.Dimen, error) {
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
func distributeMarginSpace(c boxtree.Container, w, enclosing dimen.Dimen) *frame.Box {
	box := &frame.Box{}
	box.H = c.CSSBox().H
	box.TopL = c.CSSBox().TopL
	remaining := enclosing - w
	if remaining == 0 { // TODO fit into general case
		box.SetWidth(w)
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
		box.SetWidth(w)
	}
	return box
}
