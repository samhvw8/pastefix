package main

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// Runs only with PASTEFIX_CLIPBOARD_TEST=1, because it overwrites the clipboard.
// Alfred runs scripts with no LANG; without it pbpaste turns "│" into "?".
func TestClipboardRoundTripWithoutLocale(t *testing.T) {
	if runtime.GOOS != "darwin" || os.Getenv("PASTEFIX_CLIPBOARD_TEST") != "1" {
		t.Skip("set PASTEFIX_CLIPBOARD_TEST=1 on macOS to run (overwrites the clipboard)")
	}
	for _, k := range []string{"LANG", "LC_ALL", "LC_CTYPE", "PATH"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	tb := mustParse(t, notesBox)
	box := Box(tb, 0)
	if err := writeClipboard(box, HTML(tb)); err != nil {
		t.Fatal(err)
	}
	got, err := readClipboard()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != box || !strings.Contains(got, "│ Communication") {
		t.Fatalf("clipboard changed in transit:\n%s", got)
	}
}
