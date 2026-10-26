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

var errNoClipboard = errors.New("the clipboard needs macOS; pipe the table in instead (tablefix md < table.txt)")

func readClipboard() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", errNoClipboard
	}
	out, err := exec.Command("pbpaste").Output()
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
		cmd := exec.Command("textutil", "-stdin", "-format", "html", "-inputencoding", "UTF-8", "-convert", "rtf", "-stdout")
		cmd.Stdin = strings.NewReader(richHTML)
		rtf, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("textutil: %w", err)
		}
		f, err := os.CreateTemp("", "tablefix-*.rtf")
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
	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", pasteboardJXA, plain, richHTML, rtfPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
