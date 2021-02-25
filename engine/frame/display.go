package frame

import "bytes"

// DisplayMode is a type for CSS property "display".
type DisplayMode uint16

// Flags for box context and display mode (outer and inner).
//go:generate stringer -type=DisplayMode
const (
	NoMode       DisplayMode = iota   // unset or error condition
	DisplayNone  DisplayMode = 0x0001 // CSS outer display = none
	FlowMode     DisplayMode = 0x0002 // CSS inner display = flow
	BlockMode    DisplayMode = 0x0004 // CSS block context (inner or outer)
	InlineMode   DisplayMode = 0x0008 // CSS inline context
	ListItemMode DisplayMode = 0x0010 // CSS list-item display
	FlowRoot     DisplayMode = 0x0020 // CSS flow-root display property
	FlexMode     DisplayMode = 0x0040 // CSS inner display = flex
	GridMode     DisplayMode = 0x0080 // CSS inner display = grid
	TableMode    DisplayMode = 0x0100 // CSS table display property (inner or outer)
	ContentsMode DisplayMode = 0x0200 // CSS contents display mode, experimental !
)

var allDisplayModes = []DisplayMode{
	DisplayNone, FlowMode, BlockMode, InlineMode, ListItemMode, FlowRoot, FlexMode,
	GridMode, TableMode, ContentsMode,
}

// Set sets a given atomic mode within this display mode.
func (disp *DisplayMode) Set(d DisplayMode) {
	*disp = (*disp) | d
}

// Contains checks if a display mode contains a given atomic mode.
// Returns false for d = NoMode.
func (disp DisplayMode) Contains(d DisplayMode) bool {
	return d != NoMode && (disp&d > 0)
}

// Overlaps returns true if a given display mode shares at least one atomic
// mode flag with disp (excluding NoMode).
func (disp DisplayMode) Overlaps(d DisplayMode) bool {
	for _, m := range allDisplayModes {
		if disp.Contains(m) && d.Contains(m) {
			return true
		}
	}
	return false
}

// FullString returns all atomic modes set in a display mode.
func (disp DisplayMode) FullString() string {
	var b bytes.Buffer
	first := true
	for _, m := range allDisplayModes {
		if disp.Contains(m) {
			if !first {
				b.WriteString(" ")
			}
			first = false
			b.WriteString(m.String())
		}
	}
	return b.String()
}

// Symbol returns a Unicode symbol for a mode.
func (disp DisplayMode) Symbol() string {
	if disp == FlowMode {
		return "\u25a7"
	} else if disp.Contains(BlockMode) {
		return "\u25a9"
	} else if disp.Contains(InlineMode) {
		return "\u25ba"
	} else if disp.Contains(FlexMode) {
		return "\u25a4"
	} else if disp.Contains(GridMode) {
		return "\u25f0"
	} else if disp.Contains(ListItemMode) {
		return "\u25a3"
	} else if disp.Contains(TableMode) {
		return "\u25a5"
	}
	return "?"
}
