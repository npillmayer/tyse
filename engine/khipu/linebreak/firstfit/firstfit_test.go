package firstfit

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/parameters"
	"github.com/npillmayer/tyse/engine/khipu"
	"github.com/npillmayer/tyse/engine/khipu/linebreak"
)

var graphviz = false // global switch for GraphViz DOT output

func TestBuffer(t *testing.T) {
	//gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	lb, _ := newTestLinebreaker(t, "Hello World!", 20)
	k, ok := lb.peek()
	if !ok || k.Type() != khipu.KTTextBox {
		//t.Logf("lb.pos=%d, lb.mark=%d", lb.pos, lb.check)
		t.Logf("lb.pos=%d", lb.pos)
		t.Errorf("expected the first knot to be TextBox('Hello'), is %v", k)
	}
	if knot := lb.next(); k != knot {
		t.Errorf("first knot is %v, re-read knot is %v", k, knot)
	}
}

func TestBacktrack(t *testing.T) {
	//gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	lb, _ := newTestLinebreaker(t, "the quick brown fox jumps over the lazy dog.", 30)
	k := lb.next()
	lb.checkpoint()
	lb.next()
	lb.next()
	lb.next()
	knot := lb.backtrack()
	if k != knot {
		t.Errorf("remembered start knot is %v, backtracked knot is %v", k, knot)
	}
	k = lb.next()
	lb.checkpoint()
	knot = lb.backtrack()
	if k != knot {
		t.Errorf("remembered knot is %v, backtracked knot is %v", k, knot)
	}
}
func TestLinebreak(t *testing.T) {
	// gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	lb, kh := newTestLinebreaker(t, "the quick brown fox jumps over the lazy dog.", 30)
	breakpoints, err := lb.FindBreakpoints()
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(breakpoints)-1 != 2 {
		t.Errorf("exptected 'princess' to occupy 2 lines, got %d", len(breakpoints)-1)
	}
	t.Logf("# Paragraph with %d lines: %v", len(breakpoints)-1, breakpoints)
	t.Logf("    |---------+---------+---------+|")
	j := 0
	for i := 1; i < len(breakpoints); i++ {
		//	t.Logf("%3d: %s", i, kh.Text(j, breakpoints[i].Position()))
		text := kh.Text(j, breakpoints[i].Position())
		t.Logf("%3d: %-30s|", i, justify(text, 30, i%2 == 0))
		j = breakpoints[i].Position()
	}
}

var princess = `In olden times when wishing still helped one, there lived a king whose daughters were all beautiful; and the youngest was so beautiful that the sun itself, which has seen so much, was astonished whenever it shone in her face. Close by the king's castle lay a great dark forest, and under an old lime-tree in the forest was a well, and when the day was very warm, the king's child went out into the forest and sat down by the side of the cool fountain; and when she was bored she took a golden ball, and threw it up on high and caught it; and this ball was her favorite plaything.`

func TestPrincess(t *testing.T) {
	// gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer = gotestingadapter.New()
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	lb, kh := newTestLinebreaker(t, princess, 45)
	kh.AppendKnot(khipu.Penalty(-10000)) // TODO: add parfillskip
	breakpoints, err := lb.FindBreakpoints()
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(breakpoints)-1 != 14 {
		t.Errorf("exptected 'princess' to occupy 14 lines, got %d", len(breakpoints)-1)
	}
	t.Logf("# Paragraph with %d lines: %v", len(breakpoints)-1, breakpoints)
	t.Logf("     |---------+---------+---------+---------+-----|")
	j := 0
	for i := 1; i < len(breakpoints); i++ {
		//	t.Logf("%3d: %s", i, kh.Text(j, breakpoints[i].Position()))
		text := kh.Text(j, breakpoints[i].Position())
		t.Logf("%3d: %-45s|", i, justify(text, 45, i%2 == 0))
		j = breakpoints[i].Position()
	}
}

// --- Helpers ----------------------------------------------------------

func newTestLinebreaker(t *testing.T, text string, len int) (*linebreaker, *khipu.Khipu) {
	kh, cursor, _ := setupFFTest(t, text, false)
	parshape := linebreak.RectangularParShape(dimen.Dimen(len) * 10 * dimen.BP)
	lb, err := newLinebreaker(cursor, parshape, nil)
	if err != nil {
		t.Error(err)
	}
	return lb, kh
}

func setupFFTest(t *testing.T, paragraph string, hyphens bool) (*khipu.Khipu, linebreak.Cursor, io.Writer) {
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelError)
	regs := parameters.NewTypesettingRegisters()
	if hyphens {
		regs.Push(parameters.P_MINHYPHENLENGTH, 3) // allow hyphenation
	} else {
		regs.Push(parameters.P_MINHYPHENLENGTH, 100) // inhibit hyphenation
	}
	kh := khipu.KnotEncode(strings.NewReader(paragraph), nil, regs)
	if kh == nil {
		t.Errorf("no Khipu to test; input is %s", paragraph)
	}
	//kh.AppendKnot(khipu.Penalty(linebreak.InfinityMerits))
	cursor := linebreak.NewFixedWidthCursor(khipu.NewCursor(kh), 10*dimen.BP, 0)
	var dotfile io.Writer
	var err error
	if graphviz {
		dotfile, err = ioutil.TempFile(".", "firstfit-*.dot")
		if err != nil {
			t.Errorf(err.Error())
		}
	}
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	return kh, cursor, dotfile
}

// -------------------------------------------------------------

// crude implementation just for testing
func justify(text string, l int, even bool) string {
	t := strings.Trim(text, " \t\n")
	d := l - len(t)
	if d == 0 {
		return t // fit
	} else if d < 0 { // overfull box
		return text + "\u25ae"
	}
	s := strings.Fields(text)
	if len(s) == 1 {
		return text
	}
	var b bytes.Buffer
	W := 0 // length of all words
	for _, w := range s {
		W += len(w)
	}
	d = l - W // amount of WS to distribute
	ws := d / (len(s) - 1)
	r := d - ws*(len(s)-1) + 1
	b.WriteString(s[0])
	if even {
		for j := 1; j < r; j++ {
			for i := 0; i < ws+1; i++ {
				b.WriteString(" ")
			}
			b.WriteString(s[j])
		}
		for j := r; j < len(s); j++ {
			for i := 0; i < ws; i++ {
				b.WriteString(" ")
			}
			b.WriteString(s[j])
		}
	} else {
		for j := 1; j <= len(s)-r; j++ {
			for i := 0; i < ws; i++ {
				b.WriteString(" ")
			}
			b.WriteString(s[j])
		}
		for j := len(s) - r + 1; j < len(s); j++ {
			for i := 0; i < ws+1; i++ {
				b.WriteString(" ")
			}
			b.WriteString(s[j])
		}
	}
	return b.String()
}
