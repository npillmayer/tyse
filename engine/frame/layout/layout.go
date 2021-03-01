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

var ErrEnclosingWidthNotFixed error = errors.New("enclosing width not fixed")

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

// w is width of containing block.
//
// margin-left + width + margin-right = width of containing block
func SolveWidth(c boxtree.Container, enclosing css.DimenT) (*frame.Box, error) {
	var box *frame.Box
	if enclosing.IsNone() { // bottom-up approach: calulate natural width
	} else { // top-down approach: distribute available space
		if enclosing.IsRelative() {
			return nil, ErrEnclosingWidthNotFixed
		}
		width := css.SomeDimen(c.CSSBox().Width()) // TODO Width should return an option
		calc, _ := width.Match(option.Of{
			option.None:       calcWidthAsRest,
			css.Auto:          calcWidthAsRest,
			css.ContentScaled: calcNaturalWidth,
			option.Some:       takeWidth,
		})
		solve := calc.(calcFn)
		w, _ := solve(c, width, enclosing)
		box = distributeMargin(c, w, enclosing)
	}
	return box, nil
}

type calcFn func(c boxtree.Container, w, enclosing css.DimenT) (css.DimenT, error)

func takeWidth(c boxtree.Container, w, enclosing css.DimenT) (css.DimenT, error) {
	return w, nil
}

func calcNaturalWidth(c boxtree.Container, w, enclosing css.DimenT) (css.DimenT, error) {
	// TODO
	//
	return css.Dimen(), nil
}

// Spec: If 'width' is set to 'auto', any other 'auto' values become '0'
// and 'width' follows from the resulting equality.
func calcWidthAsRest(c boxtree.Container, w, enclosing css.DimenT) (css.DimenT, error) {
	left := fixedDimension(c.CSSBox().Margins[frame.Left], c, enclosing)
	c.CSSBox().Margins[frame.Left] = css.SomeDimen(left) // do not lose fixed value
	right := fixedDimension(c.CSSBox().Margins[frame.Right], c, enclosing)
	c.CSSBox().Margins[frame.Right] = css.SomeDimen(right) // do not lose fixed value
	width := enclosing.Unwrap() - left - right
	return css.SomeDimen(width), nil
}

func fixedDimension(d css.DimenT, c boxtree.Container, enclosing css.DimenT) dimen.Dimen {
	fixed, err := d.Match(option.Of{
		option.None: dimen.Zero,
		css.Initial: dimen.Zero,
		css.Auto:    dimen.Zero,
		"%":         option.Safe(fixRelativeDimension(d, c, enclosing)),
		option.Some: d.Unwrap(),
	})
	if err != nil {
		T().Errorf("layout fix relative dimen: %s", err.Error())
		return dimen.Zero
	}
	return fixed.(dimen.Dimen)
}

func fixRelativeDimension(d css.DimenT, c boxtree.Container, w css.DimenT) (dimen.Dimen, error) {
	if !d.IsRelative() {
		return d.Unwrap(), nil
	}
	enclosing := w.Unwrap()
	width, err := d.Match(option.Of{
		option.None: dimen.Zero,
		option.Some: enclosing - (d.Unwrap() * enclosing / 100),
		// TODO css.FontScaled: d.ScaleFromFont(c.DOMNode().Font()),
		// TODO css.ViewScaled: d.ScaleFromView(...)
	})
	if err != nil {
		T().Errorf("layout fix relative dimen: %s", err.Error())
		return dimen.Zero, err
	}
	return width.(dimen.Dimen), nil
}

// w and enclosing should be fixed
func distributeMargin(c boxtree.Container, w, enclosing css.DimenT) *frame.Box {
	box := &frame.Box{}
	if !w.IsNone() && !enclosing.IsNone() {
		remaining := enclosing.Unwrap() - w.Unwrap()
		if remaining == 0 {
			box = &frame.Box{}
			box.SetWidth(w.Unwrap())
			box.H = c.CSSBox().H
			box.TopL = c.CSSBox().TopL
		} else {
			left := c.CSSBox().Margins[frame.Left]
			right := c.CSSBox().Margins[frame.Right]
			left.Match(option.Of{
				css.Auto: 1,
			})
		}
	}
	return box
}
