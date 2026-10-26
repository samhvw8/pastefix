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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"
)

const bundleID = "com.github.samhvw8.tablefix"

// Fixed object ids keep the workflow identical across reinstalls.
const (
	uidFilter = "6C1E2F0A-5B7D-4E61-9A39-1F2B3C4D5E01"
	uidRun    = "6C1E2F0A-5B7D-4E61-9A39-1F2B3C4D5E02"
	uidNotify = "6C1E2F0A-5B7D-4E61-9A39-1F2B3C4D5E03"
)

var targets = []struct{ name, where, what string }{
	{"notes", "Apple Notes / Pages / Mail / Google Docs", "Real table (rich text)"},
	{"slack", "Slack", "Aligned table in a code block"},
	{"list", "Chat box (Messenger, Zalo, iMessage)", "One block per row with bullets"},
	{"md", "Markdown", "Clean, aligned pipe table"},
	{"box", "Plain text box", "Box table without code fence"},
}

func runAlfred(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: tablefix alfred install|uninstall|export|filter")
	}
	switch args[0] {
	case "filter":
		return alfredFilter(stdout)
	case "install":
		fs := flag.NewFlagSet("alfred install", flag.ContinueOnError)
		bin := fs.String("bin", "", "tablefix binary the workflow calls (default: tablefix on PATH, else this binary)")
		keyword := fs.String("keyword", "tf", "Alfred keyword")
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
			return errors.New("no tablefix workflow is installed in Alfred")
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "removed %s\n", dir)
		return nil
	case "export":
		fs := flag.NewFlagSet("alfred export", flag.ContinueOnError)
		bin := fs.String("bin", "", "tablefix binary the workflow calls (default: tablefix on PATH, else this binary)")
		keyword := fs.String("keyword", "tf", "Alfred keyword")
		out := fs.String("o", "tablefix.alfredworkflow", "output file")
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

// alfredFilter prints Script Filter JSON: the targets with the clipboard's row
// count, or one invalid item with the error, so Enter does nothing.
func alfredFilter(stdout io.Writer) error {
	type item struct {
		UID          string `json:"uid,omitempty"`
		Title        string `json:"title"`
		Subtitle     string `json:"subtitle"`
		Arg          string `json:"arg,omitempty"`
		Match        string `json:"match,omitempty"`
		Autocomplete string `json:"autocomplete,omitempty"`
		Valid        bool   `json:"valid"`
	}
	var items []item
	text, err := clipboardReader()
	var t Table
	if err == nil {
		t, err = Parse(text)
	}
	if err != nil {
		items = []item{{Title: "No table on the clipboard", Subtitle: err.Error()}}
	} else {
		for _, tg := range targets {
			items = append(items, item{
				UID: tg.name, Title: tg.name + " — " + tg.where, Arg: tg.name, Valid: true,
				Subtitle:     fmt.Sprintf("%s · %d rows on the clipboard", tg.what, len(t.Rows)),
				Match:        tg.name + " " + tg.where,
				Autocomplete: tg.name,
			})
		}
	}
	return json.NewEncoder(stdout).Encode(map[string]any{"items": items})
}

// resolveBin picks the path the workflow calls. Alfred runs scripts without the
// user's shell PATH, so the path must be absolute.
func resolveBin(bin string) (string, error) {
	if bin == "" {
		if p, err := exec.LookPath("tablefix"); err == nil {
			bin = p
		} else if p, err := os.Executable(); err == nil {
			bin = p
		}
	}
	if bin == "" {
		return "", errors.New("cannot find the tablefix binary; pass --bin")
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
		<array><dict><key>destinationuid</key><string>{{.Run}}</string><key>modifiers</key><integer>0</integer><key>modifiersubtext</key><string></string><key>vitoclose</key><false/></dict></array>
		<key>{{.Run}}</key>
		<array><dict><key>destinationuid</key><string>{{.Notify}}</string><key>modifiers</key><integer>0</integer><key>modifiersubtext</key><string></string><key>vitoclose</key><false/></dict></array>
	</dict>
	<key>createdby</key><string>samhvw8</string>
	<key>description</key><string>Turn a copied text table into a table for Notes, Slack or chat</string>
	<key>disabled</key><false/>
	<key>name</key><string>tablefix</string>
	<key>objects</key>
	<array>
		<dict>
			<key>config</key>
			<dict>
				<key>alfredfiltersresults</key><true/>
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
				<key>title</key><string>tablefix</string>
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
	<key>webaddress</key><string>https://github.com/samhvw8/tablefix</string>
</dict>
</plist>
`))

// workflowPlist renders info.plist for a workflow that calls the tablefix binary at bin.
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
		"FilterScript": q + " alfred filter",
		"RunScript":    q + ` "$1" --copy < /dev/null 2>&1 | tail -1`,
		"Readme": "Copy a Markdown or box-drawing table, type `" + keyword + "`, pick a target, press Enter, then paste with ⌘V.\n\n" +
			"Calls " + bin + ". Reinstall with `tablefix alfred install` if that path changes.\n\nhttps://github.com/samhvw8/tablefix",
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
	if out, err := exec.Command("defaults", "read", "com.runningwithcrayons.Alfred-Preferences", "syncfolder").Output(); err == nil {
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

var bundleIDRe = regexp.MustCompile(`<key>bundleid</key>\s*<string>` + regexp.QuoteMeta(bundleID) + `</string>`)

// findInstalled returns the folder of an installed tablefix workflow, or "".
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
		out, err := exec.Command("plutil", "-convert", "xml1", "-o", "-", p).Output()
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

func writeFile(path string, data []byte) error { return os.WriteFile(path, data, 0o644) }
