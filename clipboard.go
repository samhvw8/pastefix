package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var errNoClipboard = errors.New("the clipboard needs macOS; pipe the table in instead (pastefix md < table.txt)")

// macCommand runs a macOS tool by absolute path with a UTF-8 locale. Alfred runs
// scripts with no PATH guarantee and no LANG, and without a locale pbpaste
// turns every non-ASCII character (such as "│") into "?".
func macCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command("/usr/bin/"+name, args...)
	cmd.Env = append(os.Environ(), "LANG=en_US.UTF-8", "LC_ALL=en_US.UTF-8")
	return cmd
}

func readClipboard() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", errNoClipboard
	}
	out, err := macCommand("pbpaste").Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return string(out), nil
}

// The JXA script clears the pasteboard and sets every type in one go, so each
// app takes the richest type it understands. argv: plain, html, rtf-path.
const pasteboardJXA = `
ObjC.import('AppKit');
function run(argv) {
  var pb = $.NSPasteboard.generalPasteboard;
  pb.clearContents;
  if (argv[1]) pb.setStringForType($(argv[1]), 'public.html');
  if (argv[2]) pb.setDataForType($.NSData.dataWithContentsOfFile($(argv[2])), 'public.rtf');
  pb.setStringForType($(argv[0]), 'public.utf8-plain-text');
}
`

// writeClipboard puts plain text on the clipboard, plus HTML and RTF when richHTML is set.
func writeClipboard(plain, richHTML string) error {
	if runtime.GOOS != "darwin" {
		return errNoClipboard
	}
	rtfPath := ""
	if richHTML != "" {
		cmd := macCommand("textutil", "-stdin", "-format", "html", "-inputencoding", "UTF-8", "-convert", "rtf", "-stdout")
		cmd.Stdin = strings.NewReader(richHTML)
		rtf, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("textutil: %w", err)
		}
		f, err := os.CreateTemp("", "pastefix-*.rtf")
		if err != nil {
			return err
		}
		defer os.Remove(f.Name())
		if _, err := f.Write(rtf); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		rtfPath = f.Name()
	}
	cmd := macCommand("osascript", "-l", "JavaScript", "-e", pasteboardJXA, plain, richHTML, rtfPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
