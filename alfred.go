package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"text/template"
)

const bundleID = "com.github.samhvw8.pastefix"

// legacyBundleID is the id from before the rename; install updates that workflow
// in place instead of adding a second one.
const legacyBundleID = "com.github.samhvw8.tablefix"

// Fixed object ids keep the workflow identical across reinstalls.
const (
	uidFilter = "6C1E2F0A-5B7D-4E61-9A39-1F2B3C4D5E01"
	uidRun    = "6C1E2F0A-5B7D-4E61-9A39-1F2B3C4D5E02"
	uidNotify = "6C1E2F0A-5B7D-4E61-9A39-1F2B3C4D5E03"
)

type target struct{ name, where, what string }

var targets = []target{
	{"notes", "Apple Notes / Pages / Mail / Google Docs", "Real table (rich text)"},
	{"slack", "Slack", "Aligned table in a code block"},
	{"list", "Chat box (Messenger, Zalo, iMessage)", "One block per row with bullets"},
	{"md", "Markdown", "Clean, aligned pipe table"},
	{"box", "Plain text box", "Box table without code fence"},
}

func runAlfred(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: pastefix alfred install|uninstall|export|filter")
	}
	switch args[0] {
	case "filter":
		return alfredFilter(strings.Join(args[1:], " "), stdout)
	case "run":
		// The run script passes the picked item's arg; the item sets $ai.
		return alfredRun(strings.Join(args[1:], " "), os.Getenv("ai"), stderr)
	case "install":
		fs := flag.NewFlagSet("alfred install", flag.ContinueOnError)
		bin := fs.String("bin", "", "pastefix binary the workflow calls (default: pastefix on PATH, else this binary)")
		keyword := fs.String("keyword", "pf", "Alfred keyword")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return alfredInstall(*bin, *keyword, stderr)
	case "uninstall":
		dir, err := findInstalled()
		if err != nil {
			return err
		}
		if dir == "" {
			return errors.New("no pastefix workflow is installed in Alfred")
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "removed %s\n", dir)
		return nil
	case "export":
		fs := flag.NewFlagSet("alfred export", flag.ContinueOnError)
		bin := fs.String("bin", "", "pastefix binary the workflow calls (default: pastefix on PATH, else this binary)")
		keyword := fs.String("keyword", "pf", "Alfred keyword")
		out := fs.String("o", "pastefix.alfredworkflow", "output file")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		plist, err := workflowPlist(*bin, *keyword)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		z := zip.NewWriter(&buf)
		w, err := z.Create("info.plist")
		if err != nil {
			return err
		}
		if _, err := w.Write(plist); err != nil {
			return err
		}
		if err := z.Close(); err != nil {
			return err
		}
		if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "wrote %s (open it to import into Alfred)\n", *out)
		return nil
	}
	return fmt.Errorf("unknown alfred command %q (install, uninstall, export, filter)", args[0])
}

// splitQuery reads the Alfred query as an optional target, then an optional AI
// instruction. One word that starts a target name only filters ("no" -> notes);
// a target name followed by words picks it and makes the words the instruction;
// words that name no target are the instruction for every target.
func splitQuery(query string) (names []string, instruction string) {
	query = strings.TrimSpace(query)
	first, rest, _ := strings.Cut(query, " ")
	rest = strings.TrimSpace(rest)
	var exact, prefix []string
	for _, tg := range targets {
		if strings.EqualFold(tg.name, first) {
			exact = append(exact, tg.name)
		} else if first != "" && strings.HasPrefix(tg.name, strings.ToLower(first)) {
			prefix = append(prefix, tg.name)
		}
	}
	switch {
	case query == "":
		return nil, ""
	case len(exact) == 1:
		return exact, rest
	case rest == "" && len(prefix) > 0:
		return prefix, ""
	}
	return nil, query
}

type alfredItem struct {
	UID          string `json:"uid,omitempty"`
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle"`
	Arg          string `json:"arg,omitempty"`
	Autocomplete string `json:"autocomplete,omitempty"`
	Valid        bool   `json:"valid"`
	Mods         any    `json:"mods,omitempty"`
	Variables    any    `json:"variables,omitempty"`
}

// slashCommands are the Alfred settings commands. Each maps to a CLI command.
var slashCommands = []struct{ syntax, does string }{
	{"/config", "List every setting"},
	{"/set apikey KEY", "Call the Anthropic API with this key (kept in the keychain)"},
	{"/unset apikey", "Remove the API key; use your claude login again"},
	{"/set model NAME", "Model for the AI step: haiku, sonnet, opus or a model id"},
	{"/unset model", "Go back to the default model (haiku)"},
	{"/prompt add NAME TEXT", "Save a shortcut: pf slack NAME runs TEXT"},
	{"/prompt rm NAME", "Remove a shortcut"},
}

// slashWords turns an Alfred /command into the CLI words configCommand takes.
func slashWords(query string) []string {
	w := strings.Fields(strings.TrimPrefix(strings.TrimSpace(query), "/"))
	if len(w) == 0 {
		return nil
	}
	switch w[0] {
	case "set", "unset":
		return append([]string{"config"}, w...)
	case "config", "prompt":
		return w
	}
	return nil
}

// describeCommand says what a complete command will do, without doing it, or
// returns ok=false when the words are not a complete, valid command yet.
func describeCommand(w []string) (string, bool) {
	switch {
	case len(w) == 4 && w[1] == "set" && w[2] == "apikey":
		return "Save the API key " + maskKey(w[3]) + " in the keychain", true
	case len(w) == 4 && w[1] == "set" && w[2] == "model":
		return "Use the model " + w[3] + " for the AI step", true
	case len(w) == 3 && w[1] == "unset" && (w[2] == "apikey" || w[2] == "model"):
		return "Remove the " + w[2] + " setting", true
	case len(w) >= 4 && w[0] == "prompt" && w[1] == "add" && promptNameError(w[2]) == nil:
		return "Save shortcut " + w[2] + ": " + strings.Join(w[3:], " "), true
	case len(w) == 3 && w[0] == "prompt" && w[1] == "rm":
		return "Remove shortcut " + w[2], true
	}
	return "", false
}

// commandItems answers a query that starts with "/".
func commandItems(query string) []alfredItem {
	w := slashWords(query)
	if len(w) == 1 && (w[0] == "config" || w[0] == "prompt") {
		c, err := loadConfig()
		if err != nil {
			return []alfredItem{{Title: "Cannot read the config", Subtitle: err.Error()}}
		}
		var items []alfredItem
		for _, l := range configLines(c) {
			items = append(items, alfredItem{Title: clip(l.value, 90), Subtitle: l.key})
		}
		return items
	}
	if does, ok := describeCommand(w); ok {
		return []alfredItem{{Title: does, Subtitle: "Enter to save", Arg: strings.TrimSpace(query), Valid: true}}
	}
	var items []alfredItem
	q := strings.TrimSpace(query)
	for _, c := range slashCommands {
		if strings.HasPrefix(c.syntax, q) || strings.HasPrefix(q, c.syntax) {
			auto, _, _ := strings.Cut(c.syntax, " NAME")
			auto, _, _ = strings.Cut(auto, " KEY")
			items = append(items, alfredItem{Title: c.syntax, Subtitle: c.does, Autocomplete: auto + " "})
		}
	}
	if items == nil {
		items = append(items, alfredItem{Title: "Unknown command " + q, Subtitle: "Type / to see the commands"})
	}
	return items
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

// alfredRun runs the item Alfred picked: a /command, or a target with the AI
// step in ai. The last line it writes becomes the notification.
func alfredRun(arg, ai string, notice io.Writer) error {
	if strings.HasPrefix(arg, "/") {
		w := slashWords(arg)
		if w == nil {
			return fmt.Errorf("unknown command %q", arg)
		}
		msg, err := configCommand(w, io.Discard)
		if msg != "" {
			fmt.Fprintln(notice, "pastefix: "+msg)
		}
		return err
	}
	args := []string{arg, "--copy"}
	if ai != "" {
		args = append(args, "--ai", ai)
	}
	return run(args, os.Stdin, io.Discard, notice)
}

// alfredFilter prints Script Filter JSON: the targets with the clipboard's row
// count, or one invalid item with the error, so Enter does nothing. A query that
// starts with "/" is a settings command instead.
func alfredFilter(query string, stdout io.Writer) error {
	type item = alfredItem
	if strings.HasPrefix(strings.TrimSpace(query), "/") {
		return json.NewEncoder(stdout).Encode(map[string]any{"items": commandItems(query)})
	}
	var items []item
	cfg, err := loadConfig()
	var t Table
	if err == nil {
		var text string
		if text, err = clipboardReader(); err == nil {
			t, err = Parse(text)
		}
	}
	if err != nil {
		items = []item{{Title: "No table on the clipboard", Subtitle: err.Error()}}
	} else {
		names, instruction := splitQuery(query)
		label := instruction
		if text, ok := aiSteps[instruction]; ok {
			label = instruction + ": " + text
		} else if text, ok := cfg.Prompts[instruction]; ok {
			label = instruction + ": " + text
		}
		for _, tg := range targets {
			if names != nil && !slices.Contains(names, tg.name) {
				continue
			}
			it := item{
				UID: tg.name, Title: tg.name + " — " + tg.where, Arg: tg.name, Valid: true,
				Subtitle:     fmt.Sprintf("%s · %d rows on the clipboard", tg.what, len(t.Rows)),
				Autocomplete: tg.name + " ",
				// ⌥-Enter adds the AI step; the run script reads the "ai" variable.
				Mods: map[string]any{"alt": map[string]any{
					"valid": true, "arg": tg.name, "variables": map[string]string{"ai": "clean"},
					"subtitle": "⌥ AI clean: join wrapped words, fix typos, add a missing header (a few seconds)",
				}},
			}
			if instruction != "" {
				it.Variables = map[string]string{"ai": instruction}
				it.Subtitle = fmt.Sprintf("AI %s · %d rows (a few seconds)", label, len(t.Rows))
			}
			items = append(items, it)
		}
		// With no query, list the shortcuts; Tab puts the name in the query.
		if strings.TrimSpace(query) == "" {
			for _, name := range slices.Sorted(maps.Keys(cfg.Prompts)) {
				items = append(items, item{
					Title: name + " → " + cfg.Prompts[name], Autocomplete: name,
					Subtitle: "Shortcut · Tab, then Enter on a target",
				})
			}
		}
	}
	return json.NewEncoder(stdout).Encode(map[string]any{"items": items})
}

// resolveBin picks the path the workflow calls. Alfred runs scripts without the
// user's shell PATH, so the path must be absolute.
func resolveBin(bin string) (string, error) {
	if bin == "" {
		if p, err := exec.LookPath("pastefix"); err == nil {
			bin = p
		} else if p, err := os.Executable(); err == nil {
			bin = p
		}
	}
	if bin == "" {
		return "", errors.New("cannot find the pastefix binary; pass --bin")
	}
	abs, err := filepath.Abs(bin)
	if err != nil {
		return "", err
	}
	if fi, err := os.Stat(abs); err != nil || fi.IsDir() {
		return "", fmt.Errorf("--bin %s is not a file", abs)
	}
	return abs, nil
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func xmlText(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s)) //nolint:errcheck // strings.Builder never fails
	return b.String()
}

var plistTmpl = template.Must(template.New("plist").Funcs(template.FuncMap{"x": xmlText}).Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>bundleid</key><string>{{x .BundleID}}</string>
	<key>category</key><string>Productivity</string>
	<key>connections</key>
	<dict>
		<key>{{.Filter}}</key>
		<array>
			<dict><key>destinationuid</key><string>{{.Run}}</string><key>modifiers</key><integer>0</integer><key>modifiersubtext</key><string></string><key>vitoclose</key><false/></dict>
			<dict><key>destinationuid</key><string>{{.Run}}</string><key>modifiers</key><integer>524288</integer><key>modifiersubtext</key><string></string><key>vitoclose</key><false/></dict>
		</array>
		<key>{{.Run}}</key>
		<array><dict><key>destinationuid</key><string>{{.Notify}}</string><key>modifiers</key><integer>0</integer><key>modifiersubtext</key><string></string><key>vitoclose</key><false/></dict></array>
	</dict>
	<key>createdby</key><string>samhvw8</string>
	<key>description</key><string>Turn a copied text table into a table for Notes, Slack or chat</string>
	<key>disabled</key><false/>
	<key>name</key><string>pastefix</string>
	<key>objects</key>
	<array>
		<dict>
			<key>config</key>
			<dict>
				<key>alfredfiltersresults</key><false/>
				<key>alfredfiltersresultsmatchmode</key><integer>0</integer>
				<key>argumenttreatemptyqueryasnil</key><true/>
				<key>argumenttrimmode</key><integer>0</integer>
				<key>argumenttype</key><integer>1</integer>
				<key>escaping</key><integer>102</integer>
				<key>keyword</key><string>{{x .Keyword}}</string>
				<key>queuedelaycustom</key><integer>3</integer>
				<key>queuedelayimmediatelyinitially</key><true/>
				<key>queuedelaymode</key><integer>0</integer>
				<key>queuemode</key><integer>1</integer>
				<key>runningsubtext</key><string>Reading clipboard…</string>
				<key>script</key><string>{{x .FilterScript}}</string>
				<key>scriptargtype</key><integer>1</integer>
				<key>scriptfile</key><string></string>
				<key>subtext</key><string>Convert the copied table for Notes, Slack or chat</string>
				<key>title</key><string>Fix copied table</string>
				<key>type</key><integer>0</integer>
				<key>withspace</key><true/>
			</dict>
			<key>type</key><string>alfred.workflow.input.scriptfilter</string>
			<key>uid</key><string>{{.Filter}}</string>
			<key>version</key><integer>3</integer>
		</dict>
		<dict>
			<key>config</key>
			<dict>
				<key>concurrently</key><false/>
				<key>escaping</key><integer>102</integer>
				<key>script</key><string>{{x .RunScript}}</string>
				<key>scriptargtype</key><integer>1</integer>
				<key>scriptfile</key><string></string>
				<key>type</key><integer>0</integer>
			</dict>
			<key>type</key><string>alfred.workflow.action.script</string>
			<key>uid</key><string>{{.Run}}</string>
			<key>version</key><integer>2</integer>
		</dict>
		<dict>
			<key>config</key>
			<dict>
				<key>lastpathcomponent</key><false/>
				<key>onlyshowifquerypopulated</key><true/>
				<key>removeextension</key><false/>
				<key>text</key><string>{query}</string>
				<key>title</key><string>pastefix</string>
			</dict>
			<key>type</key><string>alfred.workflow.output.notification</string>
			<key>uid</key><string>{{.Notify}}</string>
			<key>version</key><integer>1</integer>
		</dict>
	</array>
	<key>readme</key><string>{{x .Readme}}</string>
	<key>uidata</key>
	<dict>
		<key>{{.Filter}}</key><dict><key>xpos</key><integer>50</integer><key>ypos</key><integer>50</integer></dict>
		<key>{{.Run}}</key><dict><key>xpos</key><integer>250</integer><key>ypos</key><integer>50</integer></dict>
		<key>{{.Notify}}</key><dict><key>xpos</key><integer>450</integer><key>ypos</key><integer>50</integer></dict>
	</dict>
	<key>version</key><string>{{x .Version}}</string>
	<key>webaddress</key><string>https://github.com/samhvw8/pastefix</string>
</dict>
</plist>
`))

// workflowPlist renders info.plist for a workflow that calls the pastefix binary at bin.
func workflowPlist(bin, keyword string) ([]byte, error) {
	if keyword == "" || strings.ContainsAny(keyword, " \t\n") {
		return nil, fmt.Errorf("invalid --keyword %q", keyword)
	}
	bin, err := resolveBin(bin)
	if err != nil {
		return nil, err
	}
	q := shellQuote(bin)
	var buf bytes.Buffer
	err = plistTmpl.Execute(&buf, map[string]string{
		"BundleID": bundleID, "Keyword": keyword, "Version": version,
		"Filter": uidFilter, "Run": uidRun, "Notify": uidNotify,
		"FilterScript": claudeEnv() + q + ` alfred filter "$1"`,
		"RunScript":    claudeEnv() + q + ` alfred run "$1" < /dev/null 2>&1 | tail -1`,
		"Readme": "Copy a Markdown or box-drawing table, type `" + keyword + "`, pick a target, press Enter, then paste with ⌘V. " +
			"Press ⌥-Enter instead to add the AI clean step. Type an instruction after the target " +
			"(`" + keyword + " slack translate to Vietnamese`), or on its own, to rewrite the table with it. " +
			"The AI steps need the claude CLI.\n\n" +
			"Calls " + bin + ". Reinstall with `pastefix alfred install` if that path changes.\n\nhttps://github.com/samhvw8/pastefix",
	})
	return buf.Bytes(), err
}

// alfredWorkflowsDir returns Alfred's workflows folder, following a sync folder when one is set.
func alfredWorkflowsDir() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", errors.New("Alfred is macOS only")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	prefs := filepath.Join(home, "Library", "Application Support", "Alfred", "Alfred.alfredpreferences")
	if out, err := macCommand("defaults", "read", "com.runningwithcrayons.Alfred-Preferences", "syncfolder").Output(); err == nil {
		if sync := strings.TrimSpace(string(out)); sync != "" {
			if strings.HasPrefix(sync, "~/") {
				sync = filepath.Join(home, sync[2:])
			}
			prefs = filepath.Join(sync, "Alfred.alfredpreferences")
		}
	}
	dir := filepath.Join(prefs, "workflows")
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("Alfred workflows folder not found at %s (is Alfred installed with the Powerpack?)", dir)
	}
	return dir, nil
}

var bundleIDRe = regexp.MustCompile(`<key>bundleid</key>\s*<string>(` + regexp.QuoteMeta(bundleID) + `|` + regexp.QuoteMeta(legacyBundleID) + `)</string>`)

// findInstalled returns the folder of an installed pastefix workflow, or "".
func findInstalled() (string, error) {
	dir, err := alfredWorkflowsDir()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name(), "info.plist")
		if _, err := os.Stat(p); err != nil {
			continue
		}
		// plutil reads both XML and binary plists.
		out, err := macCommand("plutil", "-convert", "xml1", "-o", "-", p).Output()
		if err == nil && bundleIDRe.Match(out) {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", nil
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck // crypto/rand.Read never fails on supported platforms
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return strings.ToUpper(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))
}

func alfredInstall(bin, keyword string, stderr io.Writer) error {
	plist, err := workflowPlist(bin, keyword)
	if err != nil {
		return err
	}
	target, err := findInstalled()
	if err != nil {
		return err
	}
	action := "updated"
	if target == "" {
		dir, _ := alfredWorkflowsDir()
		target = filepath.Join(dir, "user.workflow."+newUUID())
		action = "installed"
		if err := os.Mkdir(target, 0o755); err != nil {
			return err
		}
	}
	if err := writeFile(filepath.Join(target, "info.plist"), plist); err != nil {
		return err
	}
	// Files from the Python-era workflow, which bundled its own copy.
	for _, stale := range []string{"filter.py", "tablefix.py"} {
		os.Remove(filepath.Join(target, stale))
	}
	resolved, _ := resolveBin(bin)
	fmt.Fprintf(stderr, "%s Alfred workflow (keyword %q) at %s\ncalls %s\n", action, keyword, target, resolved)
	return nil
}

// claudeEnv passes the claude CLI's path to the workflow, which has no shell PATH.
// Without claude the workflow still works; only ⌥-Enter reports the missing CLI.
func claudeEnv() string {
	if p, err := claudeBin(); err == nil {
		return "PASTEFIX_CLAUDE=" + shellQuote(p) + " "
	}
	return ""
}

func writeFile(path string, data []byte) error { return os.WriteFile(path, data, 0o644) }
