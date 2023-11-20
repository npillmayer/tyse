package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	pterm.Info.Println("Quit with <ctrl>D")   // inform user how to stop the CLI
	tracer().SetTraceLevel(tracing.LevelInfo) // will set the correct level later
	intp.REPL()                               // go into interactive mode
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
	code int
	arg  string
}

type Command struct {
	count int
	op    [32]Op
}

const (
	NOOP     = -1
	QUIT     = 0
	NAVIGATE = 1
	TABLE    = 2
	LIST     = 3
	SCRIPTS  = 4
)

func (intp *Intp) parseCommand(line string) (*Command, error) {
	command := &Command{}
	args := strings.Split(line, " ")
	command.count = len(args)
	for i, arg := range args {
		command.op[i].arg = ""
		switch arg {
		case "quit":
			command.op[i].code = QUIT
		case "->": // navigate
			command.op[i].code = NAVIGATE
		default:
			c := strings.Split(arg, ":")
			command.op[i].arg = c[1]
			switch c[0] {
			case "table":
				command.op[i].code = TABLE
				tracer().Infof("op: looking for table '%s'", c[1])
			case "map":
			case "list":
			case "ScriptList":
				command.op[i].code = SCRIPTS
				tracer().Infof("op: looking for script '%s'", c[1])
			}
		}
	}
	return command, nil
}

func (intp *Intp) execute(cmd *Command) (error, bool) {
	tracer().Infof("cmd = %v", cmd.op)
	if cmd.op[0].code == QUIT {
		return nil, true
	}
	for _, c := range cmd.op {
		switch c.code {
		case NAVIGATE:
			if intp.table == nil {
				pterm.Error.Println("cannot walk without table set")
			} else if intp.table == intp.lastPathNode().table {
				tracer().Infof("ignoring '->'")
			} else if intp.lastPathNode().location == nil {
				pterm.Error.Println("no location node to walk to")
			} else {
				if c.arg == "" {
					l := intp.lastPathNode().link
					if l == nil {
						tracer().Infof("optional link is null")
					}
					loc := l.Navigate()
					intp.stack = append(intp.stack, pathNode{location: loc})
					tracer().Infof("landed at %s", loc.Name())
				} else {
					nav := intp.lastPathNode().location
					l := nav.Map().LookupTag(ot.T(c.arg))
					if l.IsNull() {
						tracer().Errorf("lookup returns null")
					}
					n := pathNode{location: nav, link: l}
					intp.stack = append(intp.stack, n)
				}
			}
		case TABLE:
			tag := c.arg
			intp.table = intp.font.Table(ot.T(tag))
			intp.stack = intp.stack[:0]
			intp.stack = append(intp.stack, pathNode{table: intp.table})
			tracer().Infof("setting table: %v", tag)
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
			pterm.Printfln("scripts: %v", s.Map().AsTagRecordMap().Tags())
			intp.stack = append(intp.stack, pathNode{location: s})
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
