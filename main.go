// Command tablefix turns a Markdown or box-drawing text table into a table that
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

const version = "0.2.0"

const usage = `tablefix: turn a text table into one that pastes well elsewhere.

Usage:
  tablefix [target] [flags]
  tablefix alfred install [--bin PATH] [--keyword tf]
  tablefix alfred uninstall | export [-o FILE] | filter

Targets:
  notes  rich-text table (HTML + RTF) -> native table in Apple Notes, Pages,
         Mail, Google Docs, Notion, Word (default)
  slack  box table inside a code block (Slack messages have no tables)
  list   one block per row, for chat boxes (Messenger, Zalo, iMessage, Slack)
  box    box table without the code fence
  md     clean, aligned Markdown table

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
			fmt.Fprintln(os.Stderr, "tablefix:", err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) > 0 && args[0] == "alfred" {
		return runAlfred(args[1:], stdout, stderr)
	}

	target, positional := "notes", false
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		target, args, positional = args[0], args[1:], true
	}
	fs := flag.NewFlagSet("tablefix", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage); fs.PrintDefaults() }
	header := fs.String("header", "", "comma-separated column names (overrides detection)")
	noHeader := fs.Bool("no-header", false, "treat the first row as data")
	wrapAt := fs.Int("width", 40, "wrap box cells at N columns (0 = no wrap)")
	bold := fs.String("bold", "*", `bold marker for list titles: "*" Slack, "**" Markdown, "" none`)
	toStdout := fs.Bool("stdout", false, "print instead of copying to the clipboard")
	toClip := fs.Bool("copy", false, "copy to the clipboard even when stdout is not a terminal")
	showVersion := fs.Bool("version", false, "print the version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, "tablefix", version)
		return nil
	}
	if fs.NArg() > 0 { // target after the flags: tablefix --width 30 slack
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
	fmt.Fprintf(stderr, "\ntablefix: copied %d rows as %s. Paste with ⌘V.\n", len(t.Rows), kind)
	return nil
}
