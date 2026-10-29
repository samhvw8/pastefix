package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The usage log records one JSON line per run, so cuts rest on real use, not
// guesses. It is temporary: after usageLogEnd it stops writing, and stats asks for
// the cut decision. Then delete this file's logger.
const usageLogEnd = "2026-11-28"

var now = time.Now // swapped in tests

type usageRow struct {
	At         string `json:"at"`
	Target     string `json:"target"`
	Input      string `json:"input,omitempty"` // pipe, box, wrapped, none (no table)
	Exit       int    `json:"exit"`
	Via        string `json:"via"`          // cli or alfred
	AI         string `json:"ai,omitempty"` // clean, a shortcut name, or custom
	AIFallback bool   `json:"ai_fallback,omitempty"`
}

// usagePath and statePath sit next to the config file, so tests that set
// PASTEFIX_CONFIG never touch the real ones.
func usagePath() (string, error) { return besideConfig("usage.jsonl") }
func statePath() (string, error) { return besideConfig("state.json") }

func besideConfig(name string) (string, error) {
	p, err := configPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p), name), nil
}

func logEnded() bool { return now().Format(time.DateOnly) > usageLogEnd }

// logUsage appends r. A failure to log never fails the run; it says so on warn.
func logUsage(r usageRow, warn io.Writer) {
	if logEnded() {
		return
	}
	r.At = now().Format(time.RFC3339)
	err := func() error {
		p, err := usagePath()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		b, _ := json.Marshal(r)
		if _, err := f.Write(append(b, '\n')); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	}()
	if err != nil {
		fmt.Fprintf(warn, "pastefix: usage log: %v\n", err)
	}
}

// inputKind says what Parse read: a pipe table, a box table, or a box table
// with cells wrapped over several lines. Call it only on text that parsed.
func inputKind(text string) string {
	lines := tableLines(text)
	boxed := false
	for _, l := range lines {
		if ruleKind(l) == "box" {
			boxed = true
			break
		}
	}
	if !boxed {
		return "pipe"
	}
	run := 0 // data lines since the last rule
	for _, l := range lines {
		if ruleKind(l) != "" || !strings.Contains(l, "|") {
			run = 0
			continue
		}
		if strings.Trim(l, "| \t") == "" {
			continue
		}
		if run++; run > 1 {
			return "wrapped"
		}
	}
	return "box"
}

// The last target copied to the clipboard; pastefix with no target reuses it.
type state struct {
	Last string `json:"last"`
}

func lastTarget() (string, error) {
	p, err := statePath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var s state
	if err := json.Unmarshal(b, &s); err != nil {
		return "", fmt.Errorf("%s: %w", p, err)
	}
	return s.Last, nil
}

func saveLastTarget(target string) error {
	p, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, _ := json.Marshal(state{Last: target})
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// statsCommand prints use per target from the usage log, and marks targets
// nobody used as cut candidates. It only reports; the owner makes the cuts.
func statsCommand(out io.Writer) error {
	p, err := usagePath()
	if err != nil {
		return err
	}
	var rows []usageRow
	f, err := os.Open(p)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return err
	default:
		sc := bufio.NewScanner(f)
		for n := 1; sc.Scan(); n++ {
			var r usageRow
			if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
				f.Close()
				return fmt.Errorf("%s line %d: %w", p, n, err)
			}
			rows = append(rows, r)
		}
		f.Close()
		if err := sc.Err(); err != nil {
			return err
		}
	}

	if logEnded() {
		fmt.Fprintf(out, "The usage log ended %s. Decide the cuts below, then delete the logger (usage.go).\n", usageLogEnd)
	} else {
		end, _ := time.ParseInLocation(time.DateOnly, usageLogEnd, time.Local)
		fmt.Fprintf(out, "Usage log: %d runs, ends %s (%d days left).\n", len(rows), usageLogEnd, int(time.Until(end).Hours()/24)+1)
	}
	fmt.Fprintf(out, "File: %s\n\n", p)

	type count struct {
		ok, failed int
		last       string
	}
	per := map[string]*count{}
	inputs, via := map[string]int{}, map[string]int{}
	ai, fellBack := 0, 0
	for _, r := range rows {
		c := per[r.Target]
		if c == nil {
			c = &count{}
			per[r.Target] = c
		}
		if r.Exit == 0 {
			c.ok++
		} else {
			c.failed++
		}
		c.last = r.At[:min(len(r.At), 10)]
		inputs[r.Input]++
		via[r.Via]++
		if r.AI != "" {
			ai++
			if r.AIFallback {
				fellBack++
			}
		}
	}
	names := []string{}
	for _, tg := range targets {
		names = append(names, tg.name)
	}
	names = append(names, "config", "prompt")
	for _, name := range names {
		c := per[name]
		if c == nil {
			c = &count{last: "never"}
		}
		verdict := ""
		switch {
		case c.ok == 0 && c.failed == 0:
			verdict = "  candidate to cut"
		case c.ok == 0:
			verdict = "  fix or cut"
		}
		fmt.Fprintf(out, "%-7s %d %s, %d failed, last %s%s\n", name, c.ok, plural(c.ok, "run"), c.failed, c.last, verdict)
	}
	fmt.Fprintf(out, "\ninput: pipe %d, box %d, wrapped %d, no table %d\n", inputs["pipe"], inputs["box"], inputs["wrapped"], inputs["none"])
	fmt.Fprintf(out, "via: cli %d, alfred %d\n", via["cli"], via["alfred"])
	fmt.Fprintf(out, "AI step: %d runs, %d fell back to the code fix\n", ai, fellBack)
	return nil
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
