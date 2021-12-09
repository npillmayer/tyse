package harfbuzz_test

import (
	"fmt"
	"io"
	"strings"
	"testing"

	hb "github.com/benoitkugler/textlayout/harfbuzz"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/engine/glyphing"
	"github.com/npillmayer/tyse/engine/glyphing/harfbuzz"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/text/language"
)

func TestHBScript(t *testing.T) {
	id := "Plrd"
	script := language.MustParseScript(id)
	hb_script := harfbuzz.Script4HB(script)
	hstr := fmt.Sprintf("%x", uint32(hb_script))
	if hstr != "706c7264" {
		t.Logf("script %q: %x => %x", id, script, uint32(hb_script))
		t.Errorf("expected HB script of 706c7264, is %s", hstr)
	}
}

func TestHBLang(t *testing.T) {
	l := "de_DE"
	langT, err := language.Parse(l)
	if err != nil {
		t.Error(err)
	}
	h := harfbuzz.Lang4HB(langT)
	if h != "de-de" {
		t.Logf("Go lang = %v", langT)
		t.Logf("HB lang = %v, expected de-de", h)
		t.Fail()
	}
}

func TestHBDir(t *testing.T) {
	var d glyphing.Direction = glyphing.TopToBottom
	dir := harfbuzz.Direction4HB(d)
	if dir != hb.TopToBottom {
		t.Errorf("expected dir to be %d, is %d", hb.TopToBottom, dir)
	}
}

func TestHBShape(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.glyphs")
	defer teardown()
	//
	input := "Hello"
	text := strings.NewReader(input)
	font := loadGoFont(t)
	params := glyphing.Params{
		Font: font,
	}
	seq, err := harfbuzz.Shape(text, nil, nil, params)
	if err != nil {
		t.Error(err)
	}
	if seq.Glyphs == nil {
		t.Error("expected shaping output to be non-nil")
	}
	if len(seq.Glyphs) != len(input) {
		t.Errorf("expected %d output glyphs, have %d", len(input), len(seq.Glyphs))
	}
}

// ---------------------------------------------------------------------------

func loadGoFont(t *testing.T) *font.TypeCase {
	gofont := &font.ScalableFont{
		Fontname: "Go Sans",
		Filepath: "internal",
		Binary:   goregular.TTF,
	}
	var err error
	gofont.SFNT, err = sfnt.Parse(gofont.Binary)
	if err != nil {
		t.Fatal("cannot load Go font") // this cannot happen
	}
	typecase, err := gofont.PrepareCase(12.0)
	if err != nil {
		t.Fatal(err)
	}
	return typecase
}

// ---------------------------------------------------------------------------

func BenchmarkHBShape(b *testing.B) {
	var err error
	gofont := &font.ScalableFont{
		Fontname: "Go Sans",
		Filepath: "internal",
		Binary:   goregular.TTF,
	}
	gofont.SFNT, err = sfnt.Parse(gofont.Binary)
	if err != nil {
		b.Fatal("cannot load Go font") // this cannot happen
	}
	typecase, _ := gofont.PrepareCase(12.0)
	params := glyphing.Params{
		Font: typecase,
	}
	for i := 0; i < b.N; i++ {
		for _, line := range corpus {
			runes := runeread{runes: line}
			seq, err := harfbuzz.Shape(&runes, nil, nil, params)
			if err != nil || seq.Glyphs == nil {
				b.Fatal("expected shaping output to be non-nil")
			}
		}
	}
}

// runeread is a helper to wrap a `[]rune` into a cheap RuneReader.
type runeread struct {
	runes []rune
	pos   int
}

func (rr *runeread) ReadRune() (rune, int, error) {
	if rr.pos >= len(rr.runes) {
		return 0, 0, io.EOF
	}
	r := rr.runes[rr.pos]
	rr.pos++
	return r, 1, nil
}

var corpus = [][]rune{
	[]rune(`Im deutschen Grundgesetz ist der soziale Gedanke grundlegend verankert und sogar vor Änderungen geschützt. In politischen Diskussionen ist der Begriff bei uns durchgehend positiv besetzt, und dementsprechend wird er von Vertretern des gesamten politischen Spektrums vereinnahmt und gedeutet. Daran zeigt sich auch, dass der Begriff keineswegs einheitlich verstanden wird: Die soziale Gerechtigkeit des einen ist ungerecht aus Sicht des anderen.`),
	[]rune(`Soziale Gerechtigkeit ist nicht gleichbedeutend mit vollständiger Gleichheit. In Deutschland folgen wir im Großen und Ganzen der Denkrichtung einer sozial-liberalen Gerechtigkeit, wie sie u.a. auf John Rawls zurück geht. Dabei akzeptieren wir Ungleichheiten, wie sie durch Glück, Leistung, Genetik usw. zustande kommen, bejahen aber auch ein Recht des Staats zur Umverteilung für gesamtgesellschaftliche Ziele.`),
	[]rune(`Dieses Verständnis ist keineswegs universell; andere Gesellschaften akzentuieren den Gerechtigkeitsbegriff anders. Das angelsächsische Modell (USA, Großbritannien, Kanada, ...) verfolgt einen stärker liberitären Gedanken, während das skandinavische Modell (Schweden, Dänemark, Norwegen, ...) Gemeinschaft und Verteilung stärker betont [Merkel].`),
	[]rune(`Soziale Gerechtigkeit steht auch im Spannungsfeld mit einem anderen hohen Gut: der persönlichen Freiheit. Bürger der USA betonen eher die Freiheit von Beeinträchtigungen, und empginden Umverteilung daher als etwas, das der Freiheit zuwiderläuft. Im sozial-liberalen Modell verstehen wir die Freiheit eher als Freiheit zu Handlungen, insbesondere der umfassenden Teilhabe am öffentlichen Leben. Dieser Freiheitsbegriff lässt sich leichter mit einem staatlichen Eingriff zur Umverteilung aussöhnen. Bei Zielkonglikten stimmen die meisten Bundesbürger „im Zweifel für die Freiheit“ [Freiheitsindex].`),
	[]rune(`„Jede Gerechtigkeitstheorie fußt letzten Endes auf einer bestimmten Konzeption des erstrebenswerten Lebens in der Gemeinschaft und des angemessenen Gebrauchs unserer Freiheit, man könnte auch sagen auf einem bestimmten Menschenbild oder einer Vorstellung davon, worin die Würde des Menschen im Kern besteht. Darüber kann es in einer modernen pluralistischen Gesellschaft wohl keinen Konsens geben“ [Epbc9]. Das bedeutet, wir müssen immer wieder (im demokratischen Prozess) um eine Basis zur Verständigung ringen.`),
	[]rune(`Geschichtliche Entwicklung`),
	[]rune(`Mit dem Begriff der Gerechtigkeit befassten sich bereits Aristoteles und Platon. Für unser modernes Verständnis bahnbrechend war jedoch die Entwicklung der Idee individueller Freiheitsrechte gegenüber dem Staat im 16. Jahrhundert. Die Ständeordnung wich nach und nach anderen Gesellschaftsordnungen, in denen der Staat legitimiert werden musste, in die Freiheitsrechte des Einzelnen einzugreifen.`),
	[]rune(`„Die bis dahin nicht in Zweifel gezogene Vorstellung, dass es so etwas wie ein objektives Gemeinwohl gibt, das im Erhalt des Ganzen besteht und sozusagen unabhängig vom Willen der Individuen vorgegeben ist, verliert an Bedeutung. Stattdessen beginnt man vielfach, das Gemeinwohl als Summe oder Querschnitt der Einzelinteressen zu verstehen, aus denen es in irgendeiner Weise abgeleitet werden muss“ [Epbc9].`),
	[]rune(`Ein Ersatz der bis dahin vorausgesetzten göttlichen Ordnung kann durch das Gedankenexperiment eines Gesellschaftsvertrags gefunden werden. Insbesondere John Locke begründete ein liberales Gerechtigkeitsparadigma, das auf einem optimistischen Menschenbild beruht.`),
	[]rune(`Die geburtsbedingte Zugehörigkeit von Individuen zu Gruppen (Ständen) wurde abgelöst durch eine durchlässige Verortung in sozialen Schichten. Ein Gerechtigkeitsverständnis, das die Arbeiterbewegungen bis heute prägt, geht auf Karl Marx (1818 – 1863) zurück. „Es hat sich ein traditionelles sozialdemokratisches Gerechtigkeitsparadigma herausgebildet, das sich vor allem durch Arbeitszentriertheit (gerechter Anspruch der Arbeiter auf das Arbeitsprodukt) und Klassenoder Kollektivzentriertheit (Gerechtigkeit für die ganze Klasse statt individueller Gerechtigkeit) auszeichnet“ [Epbc9].`),
	[]rune(`Auch die katholische Kirche versuchte, ihren Beitrag zur Diskussion sozialer Gerechtigkeit zu leisten. 1891 und 1931 enstanden die päpstlichen Enzykliken, welche die katholische Soziallehre begründeten. „Eigentum verpglichtet“ ist das darin formulierte Leitmotiv, das sogar seinen Eingang in das Grundgesetz der Bundesrepublik fand (Artikel 14).`),
	[]rune(`Einen der wichtigsten Beiträge zum Diskurs über soziale Gerechtigkeit lieferte der US-amerikanische Philosph John Rawls (1921 – 2002), der die Idee des Gesellschaftsvertrags neu augleben ließ und das Leitmotiv „Gerechtigkeit als Fairness“ zwischen Freien und Gleichen verfolgte. Daraus leitet Rawls zwei Grundsätze ab [Rawls]:`),
	[]rune(`■ „Jedermann soll gleiches Recht auf das umfassendste System gleicher Grundfreiheiten haben, das mit dem gleichen System für alle anderen verträglich ist.“`),
	[]rune(`■ „Soziale und wirtschaftliche Ungleichheiten sind so zu gestalten, dass (a) vernünftigerweise zu erwarten ist, dass sie zu jedermanns Vorteil dienen, und (b) sie mit Positionen und Ämtern verbunden sind, die jedem offenstehen.“`),
	[]rune(`Rawls‘ Entwurf der Fairness ist einerseits liberal, erlaubt aber andererseits in gewissem Rahme eine Interpretation als Egalitarismus. Thomas Ebert schreibt dazu [Epbc9]:`),
	[]rune(`a) Sein Egalitarismus bleibt immer liberal: Die Gleichheitsforderungen werden stets durch die absolut vorrangigen Freiheitsrechte begrenzt.`),
	[]rune(`b) Sein Egalitarismus ist nur relativ und nicht absolut: Die Gleichheit ist kein Selbstzweck, sondern dient als Mittel zu dem Zweck, die Lage der Schwächsten zu verbessern; um dieses Zieles willen wird unter bestimmten Bedingungen auch Ungleichheit zugelassen.`),
	[]rune(`Die Theorien von Rawls stehen in enger Verbindung mit dem Prinzip der Marktwirtschaft und sind daher für den zeitgenössische Diskurs besonders relevant. Der egalitäre Aspekt löste eine Gegenbewegung aus, die u.a. in der Doktrin des Neoliberalismus einen Ausdruck fand. Diese Strömung bezweifelt, dass soziale Gerechtigkeit überhaupt ein legitimes Ziel politischen Handelns ist. Hauptkritik an Spielarten des Egalitarismus ist die Feststellung, dass in einer pluralistischen Gesellschaft jedes Ziel einer Förderung bzw. eines ginanziellen Ausgleichs willkürlich sei1 und zwangsweise in einen „paternalistischen Verteilungsdespotismus“ münde [Mbcdbe]. Dies führe allenfalls zu einer degenerierten Gleichheit, in der – in Anlehnung an Orwells Roman „Animal Farm“ – manche eben „gleicher als andere“ seien.`),
}
