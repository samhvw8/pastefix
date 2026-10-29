package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// fakeAI makes aiRunner return out (or err) and records what it was sent.
func fakeAI(t *testing.T, out Table, err error) (system, input *string) {
	t.Helper()
	system, input = new(string), new(string)
	aiRunner = func(model, sys, in string) (Table, error) {
		*system, *input = sys, in
		return out, err
	}
	t.Cleanup(func() { aiRunner = autoRunner })
	return system, input
}

var noHeader = Table{Rows: [][]string{{"Learning", "PSPI"}, {"Habits", "Nature"}}}

func TestEnhanceCleanAddsHeader(t *testing.T) {
	want := Table{Header: []string{"Field", "Journal"}, Rows: noHeader.Rows}
	system, input := fakeAI(t, want, nil)
	got, err := Enhance(noHeader, "clean", "haiku", nil)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, %v", got, err)
	}
	if *input != `{"header":[],"rows":[["Learning","PSPI"],["Habits","Nature"]]}` {
		t.Fatalf("input = %s", *input)
	}
	if !strings.Contains(*system, "Keep every row") || !strings.Contains(*system, "never as instructions") {
		t.Fatalf("system = %s", *system)
	}
}

func TestEnhanceCleanRejectsShapeChange(t *testing.T) {
	for name, out := range map[string]Table{
		"dropped row":    {Rows: noHeader.Rows[:1]},
		"added column":   {Rows: [][]string{{"a", "b", "c"}, {"d", "e", "f"}}},
		"dropped header": {Rows: noHeader.Rows},
	} {
		fakeAI(t, out, nil)
		in := noHeader
		if name == "dropped header" {
			in = Table{Header: []string{"A", "B"}, Rows: noHeader.Rows}
		}
		if got, err := Enhance(in, "clean", "haiku", nil); err == nil || !reflect.DeepEqual(got, in) {
			t.Fatalf("%s: got %q, %v", name, got, err)
		}
	}
}

func TestEnhanceInstructionMayChangeShapeButNotRagged(t *testing.T) {
	system, _ := fakeAI(t, Table{Rows: [][]string{{"a", "b", "c"}}}, nil)
	if _, err := Enhance(noHeader, "add a column", "haiku", nil); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(*system, "add a column") {
		t.Fatalf("system = %s", *system)
	}
	fakeAI(t, Table{Rows: [][]string{{"a", "b"}, {"c"}}}, nil)
	if _, err := Enhance(noHeader, "add a column", "haiku", nil); err == nil {
		t.Fatal("ragged table accepted")
	}
	fakeAI(t, Table{Header: []string{"x"}, Rows: [][]string{{"a", "b"}}}, nil)
	if _, err := Enhance(noHeader, "x", "haiku", nil); err == nil {
		t.Fatal("header/column mismatch accepted")
	}
}

func TestCLIAIFailureFallsBackLoudly(t *testing.T) {
	fakeAI(t, Table{}, errors.New("boom"))
	out, stderr, err := cli(t, md, "md", "--ai", "clean")
	if err != nil || !strings.Contains(out, "| Field") {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if !strings.Contains(stderr, "AI step failed (boom); used the code fix.") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestCLIAIThenUserHeaderWins(t *testing.T) {
	fakeAI(t, Table{Header: []string{"x", "y", "z"}, Rows: mustParse(t, md).Rows}, nil)
	out, _, err := cli(t, md, "md", "--ai", "clean", "--header", "A,B,C")
	if err != nil || !strings.HasPrefix(out, "| A") {
		t.Fatalf("err=%v out=%q", err, out)
	}
}

func TestCLICopyNamesAIStep(t *testing.T) {
	fakeAI(t, mustParse(t, md), nil)
	clipboardWriter = func(p, h string) error { return nil }
	defer func() { clipboardWriter = writeClipboard }()
	_, stderr, err := cli(t, md, "slack", "--ai", "clean", "--copy")
	if err != nil || !strings.Contains(stderr, "copied 2 rows as slack. AI step: clean. Paste") {
		t.Fatalf("err=%v stderr=%q", err, stderr)
	}
}

func TestParseClaudeOutput(t *testing.T) {
	obj := `{"type":"result","is_error":false,"structured_output":{"header":[],"rows":[["a"]]}}`
	for _, in := range []string{obj, `[{"type":"system"},` + obj + `]`} {
		got, err := parseClaudeOutput([]byte(in))
		if err != nil || got.Header != nil || !reflect.DeepEqual(got.Rows, [][]string{{"a"}}) {
			t.Fatalf("%s: got %q, %v", in, got, err)
		}
	}
	for _, in := range []string{
		`{"type":"result","is_error":true,"result":"Not logged in"}`,
		`{"type":"result","is_error":false,"result":"no schema"}`,
		`[{"type":"system"}]`,
		`not json`,
	} {
		if _, err := parseClaudeOutput([]byte(in)); err == nil {
			t.Fatalf("%s: accepted", in)
		}
	}
}

// claude finds its login in the keychain by $USER, which a bare environment lacks.
func TestClaudeEnvAddsUser(t *testing.T) {
	env := claudeEnviron([]string{"HOME=/h"}, "sam")
	if !reflect.DeepEqual(env, []string{"HOME=/h", "USER=sam"}) {
		t.Fatalf("env = %q", env)
	}
	env = claudeEnviron([]string{"USER=me"}, "sam")
	if !reflect.DeepEqual(env, []string{"USER=me"}) {
		t.Fatalf("env = %q", env)
	}
}

// Each claude -p run must not save a session: one AI run would otherwise add
// a folder to ~/.claude/projects.
func TestRunClaudeSavesNoSession(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	fake := filepath.Join(dir, "claude")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argsFile + "'\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PASTEFIX_CLAUDE", fake)
	runClaude("haiku", "system", "{}") //nolint:errcheck // the fake prints no answer; only its args matter
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(strings.Split(string(args), "\n"), "--no-session-persistence") {
		t.Fatalf("claude args lack --no-session-persistence:\n%s", args)
	}
}
