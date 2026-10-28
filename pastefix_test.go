package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const md = `| Field | Journals | Magazines |
|:---|---|---:|
| Learning | PSPI, Educational Psychology Review | Behavioral Scientist |
| Habits | Nature Human Behaviour | Harvard Business Review |
`

// What Apple Notes keeps after an AI box table is pasted: rules after every row, no header.
const notesBox = ` │ Communication & Storytelling │ PNAS, Journal of Communication │ Nautilus, Aeon │
 ├──────────────────────────────┼────────────────────────────────┼────────────────┤
 │ Improving Learning           │ PSPI                           │ American Educator │
 ├──────────────────────────────┼────────────────────────────────┼────────────────┤
 │ Habit Change                 │ Nature                         │ HBR            │
`

func mustParse(t *testing.T, s string) Table {
	t.Helper()
	tb, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return tb
}

func TestMarkdownHeaderDetected(t *testing.T) {
	tb := mustParse(t, md)
	if !reflect.DeepEqual(tb.Header, []string{"Field", "Journals", "Magazines"}) {
		t.Fatalf("header = %q", tb.Header)
	}
	if !reflect.DeepEqual(tb.Rows[1], []string{"Habits", "Nature Human Behaviour", "Harvard Business Review"}) {
		t.Fatalf("row = %q", tb.Rows[1])
	}
}

func TestNotesBoxTableHasNoHeader(t *testing.T) {
	tb := mustParse(t, notesBox)
	want := [][]string{
		{"Communication & Storytelling", "PNAS, Journal of Communication", "Nautilus, Aeon"},
		{"Improving Learning", "PSPI", "American Educator"},
		{"Habit Change", "Nature", "HBR"},
	}
	if tb.Header != nil || !reflect.DeepEqual(tb.Rows, want) {
		t.Fatalf("got %q / %q", tb.Header, tb.Rows)
	}
}

func TestMarkdownRulesAfterEveryRowMeanNoHeader(t *testing.T) {
	tb := mustParse(t, "| a | b |\n|---|---|\n| c | d |\n|---|---|\n| e | f |\n")
	if tb.Header != nil || len(tb.Rows) != 3 {
		t.Fatalf("got %q / %d rows", tb.Header, len(tb.Rows))
	}
}

func TestWrappedBoxRoundTrip(t *testing.T) {
	tb := mustParse(t, md)
	boxed := Box(tb, 12)
	if !strings.Contains(boxed, "╞") {
		t.Fatal("header rule is not heavy")
	}
	if strings.Count(boxed, "\n") <= 7 {
		t.Fatalf("expected wrapped cells:\n%s", boxed)
	}
	if back := mustParse(t, boxed); !reflect.DeepEqual(back, tb) {
		t.Fatalf("round trip:\n%q\n%q", back, tb)
	}
}

func TestASCIIGridWithEqualsHeader(t *testing.T) {
	tb := mustParse(t, "+---+---+\n| A | B |\n+===+===+\n| 1 | 2 |\n+---+---+\n| 3 | 4 |\n+---+---+\n")
	want := Table{Header: []string{"A", "B"}, Rows: [][]string{{"1", "2"}, {"3", "4"}}}
	if !reflect.DeepEqual(tb, want) {
		t.Fatalf("got %q", tb)
	}
}

func TestEscapedPipeStaysInCell(t *testing.T) {
	tb := mustParse(t, "| a \\| b | c |\n|---|---|\n| 1 | 2 |\n")
	if !reflect.DeepEqual(tb.Header, []string{"a | b", "c"}) {
		t.Fatalf("header = %q", tb.Header)
	}
	if !strings.Contains(Markdown(tb), `a \| b`) {
		t.Fatal("pipe not escaped in Markdown output")
	}
}

func TestShortRowsArePadded(t *testing.T) {
	tb := mustParse(t, "| a | b | c |\n|---|---|---|\n| 1 |\n")
	if !reflect.DeepEqual(tb.Rows, [][]string{{"1", "", ""}}) {
		t.Fatalf("rows = %q", tb.Rows)
	}
}

func TestWideCharactersAlign(t *testing.T) {
	out := strings.Split(Box(Table{Rows: [][]string{{"表", "x"}, {"ab", "y"}, {"Việt", "z"}}}, 0), "\n")
	for _, l := range out {
		if width(l) != width(out[0]) {
			t.Fatalf("misaligned:\n%s", strings.Join(out, "\n"))
		}
	}
}

func TestWrapKeepsLongWordsWhole(t *testing.T) {
	got := wrap("a supercalifragilistic word", 5)
	want := []string{"a", "supercalifragilistic", "word"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wrap = %q", got)
	}
}

func TestListUsesHeaderNamesAndBold(t *testing.T) {
	out := List(mustParse(t, md), "*")
	if !strings.HasPrefix(out, "*Learning*\n• Journals: PSPI") {
		t.Fatalf("list:\n%s", out)
	}
}

func TestHTMLEscapesCells(t *testing.T) {
	h := HTML(Table{Header: []string{"<b>"}, Rows: [][]string{{"a & b"}}})
	for _, want := range []string{"&lt;b&gt;", "a &amp; b", "<table"} {
		if !strings.Contains(h, want) {
			t.Fatalf("missing %q in %s", want, h)
		}
	}
}

// cli runs the CLI with piped stdin and stdout that is not a terminal.
func cli(t *testing.T, stdin string, args ...string) (string, string, error) {
	t.Helper()
	stdinPiped = func() bool { return true }
	stdoutTerminal = func() bool { return false }
	var out, errOut bytes.Buffer
	err := run(args, strings.NewReader(stdin), &out, &errOut)
	return out.String(), errOut.String(), err
}

func TestCLINoTableFails(t *testing.T) {
	if _, _, err := cli(t, "hello\n", "md"); !errors.Is(err, errNoTable) {
		t.Fatalf("err = %v", err)
	}
}

func TestCLIHeaderCountMismatchFails(t *testing.T) {
	if _, _, err := cli(t, md, "md", "--header", "x,y"); err == nil || !strings.Contains(err.Error(), "2 names") {
		t.Fatalf("err = %v", err)
	}
}

func TestCLITargetsPrintWhenPiped(t *testing.T) {
	for _, target := range []string{"notes", "slack", "list", "box", "md"} {
		out, _, err := cli(t, md, target)
		if err != nil || !strings.Contains(out, "Learning") {
			t.Fatalf("%s: err=%v out=%q", target, err, out)
		}
	}
}

func TestCLITargetAfterFlags(t *testing.T) {
	out, _, err := cli(t, md, "--width", "10", "slack")
	if err != nil || !strings.HasPrefix(out, "```") {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if _, _, err := cli(t, md, "slack", "list"); err == nil {
		t.Fatal("two targets accepted")
	}
	if _, _, err := cli(t, md, "csv"); err == nil {
		t.Fatal("unknown target accepted")
	}
}

func TestCLICopyWritesRichTextForNotes(t *testing.T) {
	var gotPlain, gotHTML string
	clipboardWriter = func(p, h string) error { gotPlain, gotHTML = p, h; return nil }
	defer func() { clipboardWriter = writeClipboard }()
	_, stderr, err := cli(t, md, "notes", "--copy")
	if err != nil || !strings.Contains(gotHTML, "<th") || !strings.Contains(gotPlain, "┌") {
		t.Fatalf("err=%v html=%q", err, gotHTML)
	}
	if !strings.Contains(stderr, "copied 2 rows as rich-text table") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestCLIReadsClipboardWhenStdinNotPiped(t *testing.T) {
	clipboardReader = func() (string, error) { return md, nil }
	defer func() { clipboardReader = readClipboard }()
	stdoutTerminal = func() bool { return false }
	stdinPiped = func() bool { return false }
	var out bytes.Buffer
	if err := run([]string{"md"}, strings.NewReader(""), &out, &bytes.Buffer{}); err != nil || !strings.Contains(out.String(), "Habits") {
		t.Fatalf("err=%v out=%q", err, out.String())
	}
}

func TestAlfredFilterJSON(t *testing.T) {
	defer func() { clipboardReader = readClipboard }()
	decode := func(query ...string) []map[string]any {
		var out bytes.Buffer
		if err := alfredFilter(strings.Join(query, " "), &out); err != nil {
			t.Fatal(err)
		}
		var v struct{ Items []map[string]any }
		if err := json.Unmarshal(out.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		return v.Items
	}
	clipboardReader = func() (string, error) { return md, nil }
	items := decode()
	if len(items) != 5 || items[0]["arg"] != "notes" || !strings.Contains(items[0]["subtitle"].(string), "2 rows") {
		t.Fatalf("items = %v", items)
	}
	alt := items[0]["mods"].(map[string]any)["alt"].(map[string]any)
	if alt["arg"] != "notes" || alt["variables"].(map[string]any)["ai"] != "clean" {
		t.Fatalf("alt mod = %v", alt)
	}
	if items[0]["variables"] != nil {
		t.Fatalf("plain Enter must not run the AI step: %v", items[0])
	}
	// A partial target name filters, with no AI step.
	if items := decode("no"); len(items) != 1 || items[0]["arg"] != "notes" || items[0]["variables"] != nil {
		t.Fatalf("items = %v", items)
	}
	// A target name, then words: Enter runs those words as the AI instruction.
	items = decode("slack", "translate to Vietnamese")
	if len(items) != 1 || items[0]["arg"] != "slack" ||
		items[0]["variables"].(map[string]any)["ai"] != "translate to Vietnamese" ||
		!strings.Contains(items[0]["subtitle"].(string), "translate to Vietnamese") {
		t.Fatalf("items = %v", items)
	}
	// Words that name no target: every target runs them as the instruction.
	items = decode("translate to Vietnamese")
	if len(items) != 5 {
		t.Fatalf("items = %v", items)
	}
	for _, it := range items {
		if it["variables"].(map[string]any)["ai"] != "translate to Vietnamese" {
			t.Fatalf("item = %v", it)
		}
	}
	clipboardReader = func() (string, error) { return "hello", nil }
	if items := decode(); len(items) != 1 || items[0]["valid"] != false {
		t.Fatalf("items = %v", items)
	}
}

func TestWorkflowPlistIsValidAndQuotesPath(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "it's pastefix")
	if err := exec.Command("touch", bin).Run(); err != nil {
		t.Fatal(err)
	}
	claude := "/opt/it's claude"
	t.Setenv("PASTEFIX_CLAUDE", claude)
	plist, err := workflowPlist(bin, "pf")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "info.plist")
	if err := writeFile(path, plist); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("plutil", "-lint", path).CombinedOutput(); err != nil {
		t.Fatalf("plutil: %v %s", err, out)
	}
	script, err := exec.Command("plutil", "-extract", "objects.1.config.script", "raw", path).Output()
	if err != nil {
		t.Fatal(err)
	}
	// The quoted path must survive the shell: run it through bash and compare.
	echo, err := exec.Command("bash", "-c", "printf %s "+shellQuote(bin)).Output()
	if err != nil || string(echo) != bin {
		t.Fatalf("quoting broke the path: %q", echo)
	}
	if !strings.Contains(string(script), "PASTEFIX_CLAUDE="+shellQuote(claude)+" "+shellQuote(bin)+` alfred run "$1" < /dev/null`) {
		t.Fatalf("script = %s", script)
	}
	filter, err := exec.Command("plutil", "-extract", "objects.0.config.script", "raw", path).Output()
	if err != nil || strings.TrimSpace(string(filter)) != "PASTEFIX_CLAUDE="+shellQuote(claude)+" "+shellQuote(bin)+` alfred filter "$1"` {
		t.Fatalf("filter script = %s (%v)", filter, err)
	}
	// The filter does its own matching, so Alfred must not hide items.
	if out, err := exec.Command("plutil", "-extract", "objects.0.config.alfredfiltersresults", "raw", path).Output(); err != nil || strings.TrimSpace(string(out)) != "false" {
		t.Fatalf("alfredfiltersresults = %s (%v)", out, err)
	}
	if _, err := workflowPlist(bin, "t f"); err == nil {
		t.Fatal("keyword with a space accepted")
	}
}

func TestInstallFindsWorkflowFromBeforeTheRename(t *testing.T) {
	for _, id := range []string{"com.github.samhvw8.pastefix", "com.github.samhvw8.tablefix"} {
		if !bundleIDRe.MatchString("<key>bundleid</key>\n\t<string>" + id + "</string>") {
			t.Fatalf("%s not matched", id)
		}
	}
	if bundleIDRe.MatchString("<key>bundleid</key><string>com.github.someone.else</string>") {
		t.Fatal("matched another workflow")
	}
}
