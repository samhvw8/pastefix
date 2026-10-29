// Command pastefix turns a Markdown or box-drawing text table into a table that
// pastes well elsewhere: a rich-text table for Apple Notes, a code-block table for
// Slack, or a list for chat boxes.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const version = "0.5.2"

const usage = `pastefix: turn a text table into one that pastes well elsewhere.

Usage:
  pastefix [target] [flags]
  pastefix config                          list every setting
  pastefix config set apikey|model VALUE   apikey goes to the macOS keychain
  pastefix config unset apikey|model
  pastefix prompt add NAME TEXT...         save a shortcut: --ai NAME runs TEXT
  pastefix prompt rm NAME
  pastefix alfred install [--bin PATH] [--keyword pf]
  pastefix alfred uninstall | export [-o FILE] | filter | run

Targets:
  notes  rich-text table (HTML + RTF) -> native table in Apple Notes, Pages,
         Mail, Google Docs, Notion, Word (default)
  slack  box table inside a code block (Slack messages have no tables)
  list   one block per row, for chat boxes (Messenger, Zalo, iMessage, Slack)
  box    box table without the code fence
  md     clean, aligned Markdown table

AI step (optional): after the table is fixed in code, a model edits its text.
The code still draws the table, so the model cannot break it.
  --ai clean              join words split by wrapping, fix typos, add a missing
                          header; keeps every row, column and fact
  --ai NAME               a shortcut saved with "prompt add"
  --ai "<instruction>"    anything else, e.g. --ai "translate to Vietnamese"
With an API key set, it calls the Anthropic API; else it runs the claude CLI on
your Claude login. If the AI step fails, pastefix says why and uses the table
fixed in code.

Input is stdin when it is a pipe or file, else the clipboard. Output goes to the
clipboard when run in a terminal (preview on stderr), else to stdout.

Flags:
`

// Swapped in tests.
var (
	clipboardReader = readClipboard
	clipboardWriter = writeClipboard
	stdinPiped      = func() bool {
		fi, err := os.Stdin.Stat()
		return err == nil && (fi.Mode()&os.ModeNamedPipe != 0 || fi.Mode().IsRegular())
	}
	stdoutTerminal = func() bool { return term.IsTerminal(int(os.Stdout.Fd())) }
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "pastefix:", err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) > 0 && args[0] == "alfred" {
		return runAlfred(args[1:], stdout, stderr)
	}
	if len(args) > 0 && (args[0] == "config" || args[0] == "prompt") {
		msg, err := configCommand(args, stdout)
		if msg != "" {
			fmt.Fprintln(stderr, "pastefix: "+msg)
		}
		return err
	}

	target, positional := "notes", false
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		target, args, positional = args[0], args[1:], true
	}
	fs := flag.NewFlagSet("pastefix", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage); fs.PrintDefaults() }
	header := fs.String("header", "", "comma-separated column names (overrides detection)")
	noHeader := fs.Bool("no-header", false, "treat the first row as data")
	wrapAt := fs.Int("width", 40, "wrap box cells at N columns (0 = no wrap)")
	bold := fs.String("bold", "*", `bold marker for list titles: "*" Slack, "**" Markdown, "" none`)
	toStdout := fs.Bool("stdout", false, "print instead of copying to the clipboard")
	toClip := fs.Bool("copy", false, "copy to the clipboard even when stdout is not a terminal")
	showVersion := fs.Bool("version", false, "print the version")
	aiStep := fs.String("ai", "", `AI step after the code fix: "clean" or a free instruction`)
	aiModel := fs.String("ai-model", "", "model for --ai (default: the config model, else haiku)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, "pastefix", version)
		return nil
	}
	if fs.NArg() > 0 { // target after the flags: pastefix --width 30 slack
		if positional || fs.NArg() > 1 {
			return fmt.Errorf("unexpected argument %q", fs.Arg(fs.NArg()-1))
		}
		target = fs.Arg(0)
	}
	switch target {
	case "notes", "slack", "list", "box", "md":
	default:
		return fmt.Errorf("unknown target %q (notes, slack, list, box, md)", target)
	}

	var text string
	if stdinPiped() {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		text = string(b)
	} else {
		var err error
		if text, err = clipboardReader(); err != nil {
			return err
		}
	}

	t, err := Parse(text)
	if err != nil {
		return err
	}
	aiNote := ""
	if *aiStep != "" {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		model := *aiModel
		if model == "" {
			model = cfg.Model
		}
		if model == "" {
			model = defaultModel
		}
		// A failed AI step falls back to the code fix, and says so.
		if et, err := Enhance(t, *aiStep, model, cfg.Prompts); err != nil {
			aiNote = fmt.Sprintf(" AI step failed (%v); used the code fix.", err)
			fmt.Fprintln(stderr, "pastefix:"+aiNote)
		} else {
			t, aiNote = et, " AI step: "+aiLabel(*aiStep, cfg.Prompts)+"."
		}
	}
	if *noHeader {
		t.SetHeader(nil) //nolint:errcheck // nil never fails
	}
	if *header != "" {
		names := strings.Split(*header, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		if err := t.SetHeader(names); err != nil {
			return err
		}
	}

	var out, rich string
	switch target {
	case "notes":
		out, rich = Box(t, *wrapAt), HTML(t)
	case "slack":
		out = "```\n" + Box(t, *wrapAt) + "\n```"
	case "box":
		out = Box(t, *wrapAt)
	case "list":
		out = List(t, *bold)
	case "md":
		out = Markdown(t)
	}

	if *toStdout || !(*toClip || stdoutTerminal()) {
		_, err := fmt.Fprintln(stdout, out)
		return err
	}
	if err := clipboardWriter(out, rich); err != nil {
		return err
	}
	fmt.Fprintln(stderr, out)
	kind := target
	if rich != "" {
		kind = "rich-text table"
	}
	fmt.Fprintf(stderr, "\npastefix: copied %d rows as %s.%s Paste with ⌘V.\n", len(t.Rows), kind, aiNote)
	return nil
}

// aiLabel names the step in the copy message: the built-in step or shortcut
// name, else "custom".
func aiLabel(step string, prompts map[string]string) string {
	if _, ok := aiSteps[step]; ok {
		return step
	}
	if _, ok := prompts[step]; ok {
		return step
	}
	return "custom"
}
