package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/npillmayer/schuko/schukonf/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/schuko/tracing/trace2go"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"github.com/pterm/pterm"
)

// tracer traces with key 'tyse.fonts'
func tracer() tracing.Trace {
	return tracing.Select("tyse.fonts")
}

func main() {
	initDisplay()

	// set up logging
	tracing.RegisterTraceAdapter("go", gologadapter.GetAdapter(), false)
	conf := testconfig.Conf{
		"tracing.adapter":  "go",
		"trace.tyse.fonts": "Info",
	}
	if err := trace2go.ConfigureRoot(conf, "trace", trace2go.ReplaceTracers(true)); err != nil {
		fmt.Printf("error configuring tracing")
		os.Exit(1)
	}
	tracing.SetTraceSelector(trace2go.Selector())

	// command line flags
	tlevel := flag.String("trace", "Info", "Trace level [Debug|Info|Error]")
	fontname := flag.String("font", "", "Font to load")
	flag.Parse()
	tracer().SetTraceLevel(tracing.LevelError)    // will set the correct level later
	pterm.Info.Println("Welcome to OpenType CLI") // colored welcome message
	tracer().Infof("Trace level is %s", *tlevel)
	//
	// set up REPL
	repl, err := readline.New("ot > ")
	if err != nil {
		tracer().Errorf(err.Error())
		os.Exit(3)
	}
	intp := &Intp{repl: repl, stack: make([]pathNode, 0, 100)}
	//
	// load font to use
	if err := intp.loadFont(*fontname); err != nil { // font name provided by flag
		tracer().Errorf(err.Error())
		os.Exit(4)
	}
	//
	// start receiving commands
	pterm.Info.Println("Quit with <ctrl>D") // inform user how to stop the CLI
	tracer().SetTraceLevel(tracing.LevelDebug)
	intp.REPL() // go into interactive mode
}

// We use pterm for moderately fancy output.
func initDisplay() {
	pterm.EnableDebugMessages()
	pterm.Info.Prefix = pterm.Prefix{
		Text:  " !  ",
		Style: pterm.NewStyle(pterm.BgCyan, pterm.FgBlack),
	}
	pterm.Error.Prefix = pterm.Prefix{
		Text:  " Error",
		Style: pterm.NewStyle(pterm.BgRed, pterm.FgBlack),
	}
}

type pathNode struct {
	table    ot.Table
	location ot.Navigator
	link     ot.NavLink
}

// Intp is our interpreter object
type Intp struct {
	font  *ot.Font
	repl  *readline.Instance
	table ot.Table
	stack []pathNode
}

// REPL starts interactive mode.
func (intp *Intp) REPL() {
	for {
		line, err := intp.repl.Readline()
		if err != nil { // io.EOF
			break
		}
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		println(line)
		cmd, err := intp.parseCommand(line)
		if err != nil {
			tracer().Errorf(err.Error())
			continue
		}
		err, quit := intp.execute(cmd)
		if err != nil {
			tracer().Errorf(err.Error())
			continue
		}
		if quit {
			break
		}
	}
	pterm.Info.Println("Good bye!")
}

type Op struct {
	code   int
	arg    string
	format string
}

type Command struct {
	count int
	op    [32]Op
}

const NOOP = -1
const (
	QUIT int = iota
	HELP
	NAVIGATE
	TABLE
	LIST
	MAP
	SCRIPTS
	FEATURES
)

func (intp *Intp) parseCommand(line string) (*Command, error) {
	command := &Command{}
	steps := strings.Split(line, " ")
	command.count = len(steps)
	for i, step := range steps {
		command.op[i].arg = ""
		switch step {
		case "quit":
			command.op[i].code = QUIT
		case "->": // navigate
			command.op[i].code = NAVIGATE
		default:
			c := strings.Split(step, ":") // e.g.  "scripts:latn:tag" or "list:5:int" or "help:lang" or "map"
			tracer().Infof("parse command = %v", c)
			command.op[i].arg = getOptArg(c, 1)
			command.op[i].format = getOptArg(c, 2)
			switch strings.ToLower(c[0]) {
			case "table":
				command.op[i].code = TABLE
				tracer().Infof("table: looking for table '%s'", command.op[i].arg)
			case "map":
				command.op[i].code = MAP
				tracer().Infof("map: looking for key '%v'", command.op[i].arg)
			case "list":
				command.op[i].code = LIST
				tracer().Infof("list: looking for index '%v'", command.op[i].arg)
			case "scriptlist", "scripts":
				command.op[i].code = SCRIPTS
				tracer().Infof("script-list: looking for script '%s'", command.op[i].arg)
			case "featurelist", "features":
				command.op[i].code = FEATURES
				tracer().Infof("feature-list")
			default:
				command.op[i].code = HELP
			}
		}
	}
	return command, nil
}

func (intp *Intp) execute(cmd *Command) (error, bool) {
	tracer().Infof("cmd = %v", cmd.op)
	if cmd.op[0].code == HELP {
		help(cmd.op[0].arg)
		return nil, false
	}
	if cmd.op[0].code == QUIT {
		return nil, true
	}
	for _, c := range cmd.op {
		switch c.code {
		case NAVIGATE:
			if intp.table == nil {
				pterm.Error.Println("cannot walk without table being set")
			} else if intp.table == intp.lastPathNode().table {
				tracer().Infof("ignoring '->'")
			} else if intp.lastPathNode().link == nil {
				pterm.Error.Println("no link to walk")
			} else {
				l := intp.lastPathNode().link
				n := pathNode{location: l.Navigate()}
				intp.stack = append(intp.stack, n)
				tracer().Infof("walked to %s", n.location.Name())
			}
		case TABLE:
			tag := c.arg
			intp.table = intp.font.Table(ot.T(tag))
			intp.stack = intp.stack[:0]
			intp.stack = append(intp.stack, pathNode{table: intp.table})
			tracer().Infof("setting table: %v", tag)
		case MAP:
			if intp.table == nil {
				pterm.Error.Println("cannot map without table being set")
			}
			var target ot.NavLink
			m := intp.lastPathNode().location.Map()
			if c.arg != "" {
				tag := c.arg
				if m.IsTagRecordMap() {
					trm := m.AsTagRecordMap()
					target = trm.LookupTag(ot.T(tag))
					tracer().Infof("%s map keys = %v", trm.Name(), trm.Tags())
					pterm.Printfln("%s table maps [tag %v] = %v", trm.Name(), ot.T(tag), target.Name())
				} else {
					target = m.LookupTag(ot.T(tag))
					pterm.Printfln("%s table maps [%v] = %v", m.Name(), ot.T(tag), target.Name())
				}
			} else if m.IsTagRecordMap() {
				trm := m.AsTagRecordMap()
				pterm.Printfln("%s map keys = %v", trm.Name(), trm.Tags())
			}
			n := intp.lastPathNode()
			n.link = target
			intp.setLastPathNode(n)
		case LIST:
			if intp.table == nil {
				pterm.Error.Println("cannot list without table being set")
			}
			l := intp.lastPathNode().location.List()
			if c.arg == "" {
				pterm.Printfln("List has %d entries", l.Len())
			} else if i, err := strconv.Atoi(c.arg); err == nil {
				loc := l.Get(i)
				size := loc.Size()
				value := decodeLocation(loc, l.Name())
				switch value.(type) {
				case int:
					pterm.Printfln("%s list index %d holds number = %d", l.Name(), i, value)
				default:
					pterm.Printfln("%s list index %d holds data of %d bytes", l.Name(), i, size)
				}
			} else {
				pterm.Error.Printfln("List index not numeric: %v", c.arg)
			}
		case SCRIPTS:
			if err := intp.checkTable(); err != nil {
				return err, false
			}
			s := intp.table.Self().AsGSub().ScriptList
			if s == nil {
				s = intp.table.Self().AsGPos().ScriptList
			}
			if s == nil {
				return errors.New("table has no script list"), false
			}
			m := s.Map().AsTagRecordMap()
			pterm.Printfln("ScriptList keys: %v", m.Tags())
			n := pathNode{location: s}
			if c.arg != "" {
				l := m.LookupTag(ot.T(c.arg))
				if l.IsNull() {
					tracer().Infof("script lookup [%s] returns null", ot.T(c.arg).String())
					break
				}
				n.link = l
			}
			intp.stack = append(intp.stack, n)
		case FEATURES:
			f := intp.table.Self().AsGSub().FeatureList
			if c.arg == "" {
				tracer().Infof("%s table has %d entries", f.Name(), f.Len())
			} else if i, err := strconv.Atoi(c.arg); err == nil {
				tag, _ := f.Get(i)
				//tag, lnk := f.Get(i)
				pterm.Printfln("%s list index %d holds feature record = %v", f.Name(), i, tag)
			} else {
				pterm.Error.Printfln("List index not numeric: %v", c.arg)
			}
		}
	}
	return nil, false
}

func (intp *Intp) checkTable() error {
	if intp.table == nil {
		return errors.New("not table set")
	}
	return nil
}

func (intp *Intp) loadFont(fontname string) (err error) {
	intp.font, err = loadLocalFont(fontname)
	if err == nil {
		pterm.Printfln("font tables: %v", intp.font.TableTags())
	}
	return
}

func loadLocalFont(fontFileName string) (*ot.Font, error) {
	path := filepath.Join("..", "testdata", fontFileName)
	f, err := font.LoadOpenTypeFont(path)
	if err != nil {
		tracer().Errorf("cannot load test font %s: %s", fontFileName, err)
		return nil, err
	}
	tracer().Infof("loaded SFNT font = %s", f.Fontname)
	otf, err := ot.Parse(f.Binary)
	if err != nil {
		tracer().Errorf("cannot decode test font %s: %s", fontFileName, err)
		return nil, err
	}
	otf.F = f
	tracer().Infof("parsed OpenType font = %s", otf.F.Fontname)
	return otf, nil
}

func (intp *Intp) lastPathNode() pathNode {
	if len(intp.stack) == 0 {
		return pathNode{}
	}
	return intp.stack[len(intp.stack)-1]
}

func (intp *Intp) setLastPathNode(n pathNode) {
	if len(intp.stack) == 0 {
		intp.stack = append(intp.stack, n)
	}
	intp.stack[len(intp.stack)-1] = n
}

func decodeLocation(loc ot.NavLocation, name string) interface{} {
	if loc == nil {
		return nil
	}
	switch loc.Size() {
	case 2:
		return int(loc.U16(0))
	case 4:
		return int(loc.U32(0))
	default:
		switch name {
		case "FeatureRecord":
			tag := ot.Tag(loc.U32(0))
			link := int(loc.U16(4))
			return struct {
				ot.Tag
				int
			}{tag, link}
		}
	}
	return nil
}

func help(topic string) {
	tracer().Infof("help %v", topic)
	t := strings.ToLower(topic)
	switch t {
	case "script", "scripts", "scriptList":
		pterm.Info.Println("ScriptList / Script")
		pterm.Println(`
	ScriptList is a property of GSUB and GPOS. 
	It consists of ScriptRecords:
	+------------+----------------+
	| Script Tag | Link to Script |
	+------------+----------------+
	ScriptList behaves as a map.

	A Script table links to a default LangSys entry, and contains a list of LangSys records:
	+--------------------------------+
	| Link to LangSys record         |
	+--------------+-----------------+
	| Language Tag | Link to LangSys |
	+--------------+-----------------+
	Script behaves as a map, with entry 0 as the default link
	`)
	case "lang", "langsys", "langs", "language":
		pterm.Info.Println("LangSys")
		pterm.Println(`
	LangSys is pointed to from a Script Record.
	It links a language with features to activate. It does to using an index into the feature table.
	+-----------------------------------+
	| Infex of required feature or null |
	+-----------------------------------+
	| Index of feature 1                |
	+-----------------------------------+
	| Index of feature 2                |
	+-----------------------------------+
	| ...                               |
	+-----------------------------------+
	LangSys behaves as a list.
	`)
	default:
		pterm.Info.Println("General Help, TODO")
	}
}

func getOptArg(s []string, inx int) string {
	if len(s) > inx {
		return s[inx]
	}
	return ""
}
