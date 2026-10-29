package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const wrappedBox = `┌──────┬───────────┐
│ Name │ Note      │
╞══════╪═══════════╡
│ a    │ first     │
│      │ line      │
├──────┼───────────┤
│ b    │ second    │
└──────┴───────────┘
`

const plainBox = `┌──────┬────────┐
│ Name │ Note   │
╞══════╪════════╡
│ a    │ first  │
├──────┼────────┤
│ b    │ second │
└──────┴────────┘
`

func TestInputKind(t *testing.T) {
	for in, want := range map[string]string{md: "pipe", plainBox: "box", wrappedBox: "wrapped", notesBox: "box"} {
		if got := inputKind(in); got != want {
			t.Errorf("inputKind(%.30q) = %q, want %q", in, got, want)
		}
	}
}

// usageRows returns the lines of the usage log in the current config folder.
func usageRows(t *testing.T) []string {
	t.Helper()
	p, err := usagePath()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func TestEveryConversionAddsOneUsageRow(t *testing.T) {
	freshConfig(t)
	if _, _, err := cli(t, md, "slack"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cli(t, "no table here", "list"); err == nil {
		t.Fatal("no table accepted")
	}
	cli(t, md, "--version") //nolint:errcheck // not a conversion: no row
	rows := usageRows(t)
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d: %v", len(rows), rows)
	}
	for i, want := range []string{`"target":"slack","input":"pipe","exit":0,"via":"cli"`, `"target":"list","input":"none","exit":1,"via":"cli"`} {
		if !strings.Contains(rows[i], want) {
			t.Errorf("row %d = %s, want it to contain %s", i, rows[i], want)
		}
	}
}

func TestAlfredRunIsLoggedAsAlfred(t *testing.T) {
	freshConfig(t)
	clipboardReader = func() (string, error) { return wrappedBox, nil }
	clipboardWriter = func(string, string) error { return nil }
	stdinPiped = func() bool { return false }
	t.Cleanup(func() { clipboardReader, clipboardWriter = readClipboard, writeClipboard })
	if err := alfredRun("list", "", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	rows := usageRows(t)
	if len(rows) != 1 || !strings.Contains(rows[0], `"target":"list","input":"wrapped","exit":0,"via":"alfred"`) {
		t.Fatalf("rows = %v", rows)
	}
}

func TestAIFallbackIsLogged(t *testing.T) {
	freshConfig(t)
	fakeAI(t, Table{}, os.ErrDeadlineExceeded)
	if _, _, err := cli(t, md, "md", "--ai", "clean"); err != nil {
		t.Fatal(err)
	}
	if rows := usageRows(t); len(rows) != 1 || !strings.Contains(rows[0], `"ai":"clean","ai_fallback":true`) {
		t.Fatalf("rows = %v", rows)
	}
}

func TestUsageLogStopsAfterItsEndDate(t *testing.T) {
	freshConfig(t)
	now = func() time.Time { return time.Date(2026, 11, 29, 0, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = time.Now })
	if _, _, err := cli(t, md, "md"); err != nil {
		t.Fatal(err)
	}
	if rows := usageRows(t); rows != nil {
		t.Fatalf("logged after the end date: %v", rows)
	}
	if out := runOK(t, "stats"); !strings.Contains(out, "ended 2026-11-28") {
		t.Fatalf("stats does not say the log ended:\n%s", out)
	}
}

func TestStatsMarksCutCandidates(t *testing.T) {
	freshConfig(t)
	cli(t, md, "slack")       //nolint:errcheck
	cli(t, md, "slack")       //nolint:errcheck
	cli(t, "no table", "box") //nolint:errcheck
	out := runOK(t, "stats")
	for _, want := range []string{"slack", "2 runs", "box", "fix or cut", "notes", "candidate to cut", "pipe 2"} {
		if !strings.Contains(out, want) {
			t.Errorf("stats lacks %q:\n%s", want, out)
		}
	}
}

func TestBrokenUsageLogWarnsButConverts(t *testing.T) {
	freshConfig(t)
	p, _ := usagePath()
	if err := os.MkdirAll(p, 0o755); err != nil { // a folder where the file should be
		t.Fatal(err)
	}
	out, errOut, err := cli(t, md, "md")
	if err != nil || !strings.Contains(out, "| Field") || !strings.Contains(errOut, "usage log") {
		t.Fatalf("err=%v out=%q stderr=%q", err, out, errOut)
	}
}

// pastefix with no target reuses the last target copied to the clipboard; a piped
// run (a script) always gets notes and never changes it.
func TestNoTargetUsesLastClipboardTarget(t *testing.T) {
	freshConfig(t)
	var copied string
	clipboardReader = func() (string, error) { return md, nil }
	clipboardWriter = func(plain, _ string) error { copied = plain; return nil }
	stdinPiped = func() bool { return false }
	t.Cleanup(func() { clipboardReader, clipboardWriter = readClipboard, writeClipboard })
	copyRun := func(args ...string) {
		t.Helper()
		stdinPiped = func() bool { return false }
		var errOut bytes.Buffer
		if err := run(append(args, "--copy"), strings.NewReader(""), &bytes.Buffer{}, &errOut); err != nil {
			t.Fatal(err, errOut.String())
		}
	}
	copyRun()
	if !strings.Contains(copied, "┌") { // notes: box text beside the rich table
		t.Fatalf("first default run is not notes: %q", copied)
	}
	copyRun("list")
	copyRun()
	if !strings.HasPrefix(copied, "*Learning*") {
		t.Fatalf("default run did not reuse list: %q", copied)
	}
	if out, _, err := cli(t, md); err != nil || !strings.Contains(out, "┌") {
		t.Fatalf("piped default is not notes: %v %q", err, out)
	}
	if out, _, _ := cli(t, md, "md"); !strings.Contains(out, "| Field") {
		t.Fatal(out)
	}
	copyRun()
	if !strings.HasPrefix(copied, "*Learning*") {
		t.Fatalf("a piped run changed the last target: %q", copied)
	}
	if p, _ := statePath(); filepath.Base(p) != "state.json" {
		t.Fatal(p)
	}
}
