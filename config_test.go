package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestMain keeps every test away from the real config file and keychain: a real
// API key there would make tests call the paid API.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "pastefix-test-")
	if err != nil {
		panic(err)
	}
	os.Setenv("PASTEFIX_CONFIG", filepath.Join(dir, "config.json"))
	useFakeKeychain()
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

var fakeKey string

func useFakeKeychain() {
	keychainGet = func() (string, error) { return fakeKey, nil }
	keychainSet = func(k string) error { fakeKey = k; return nil }
	keychainDelete = func() error { fakeKey = ""; return nil }
}

// freshConfig gives the test an empty config and no API key.
func freshConfig(t *testing.T) {
	t.Helper()
	t.Setenv("PASTEFIX_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	fakeKey = ""
	t.Cleanup(func() { fakeKey = "" })
}

func runOK(t *testing.T, args ...string) string {
	t.Helper()
	var out, errOut bytes.Buffer
	if err := run(args, strings.NewReader(""), &out, &errOut); err != nil {
		t.Fatalf("%v: %v (%s)", args, err, errOut.String())
	}
	return out.String() + errOut.String()
}

func TestConfigSetModelAndPrompts(t *testing.T) {
	freshConfig(t)
	runOK(t, "config", "set", "model", "sonnet")
	runOK(t, "prompt", "add", "vi", "translate", "to", "Vietnamese")
	c, err := loadConfig()
	if err != nil || c.Model != "sonnet" || c.Prompts["vi"] != "translate to Vietnamese" {
		t.Fatalf("config = %+v, %v", c, err)
	}
	out := runOK(t, "config")
	for _, want := range []string{"sonnet", "vi", "translate to Vietnamese", "clean", "claude login"} {
		if !strings.Contains(out, want) {
			t.Fatalf("config output lacks %q:\n%s", want, out)
		}
	}
	runOK(t, "prompt", "rm", "vi")
	runOK(t, "config", "unset", "model")
	if c, _ := loadConfig(); c.Model != "" || len(c.Prompts) != 0 {
		t.Fatalf("config = %+v", c)
	}
}

func TestPromptNamesThatWouldBeAmbiguousAreRejected(t *testing.T) {
	freshConfig(t)
	for _, name := range []string{"slack", "clean", "/x", ""} {
		if err := run([]string{"prompt", "add", name, "do it"}, nil, io.Discard, io.Discard); err == nil {
			t.Fatalf("prompt name %q accepted", name)
		}
	}
	if err := run([]string{"prompt", "rm", "nope"}, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("removing a missing prompt succeeded")
	}
	if err := run([]string{"config", "set", "colour", "red"}, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("unknown config key accepted")
	}
}

func TestBrokenConfigFileFailsLoudly(t *testing.T) {
	freshConfig(t)
	os.WriteFile(os.Getenv("PASTEFIX_CONFIG"), []byte("{not json"), 0o600)
	if err := run([]string{"config"}, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("broken config file was ignored")
	}
}

func TestAPIKeyGoesToKeychainAndIsMasked(t *testing.T) {
	freshConfig(t)
	key := "sk-ant-api03-secretsecret-WXYZ"
	out := runOK(t, "config", "set", "apikey", key)
	if fakeKey != key {
		t.Fatalf("keychain = %q", fakeKey)
	}
	out += runOK(t, "config")
	if strings.Contains(out, "secretsecret") || !strings.Contains(out, "…WXYZ") || !strings.Contains(out, "Anthropic API") {
		t.Fatalf("output:\n%s", out)
	}
	b, _ := os.ReadFile(os.Getenv("PASTEFIX_CONFIG"))
	if strings.Contains(string(b), "secret") {
		t.Fatalf("key written to the config file: %s", b)
	}
	runOK(t, "config", "unset", "apikey")
	if fakeKey != "" {
		t.Fatal("key not deleted")
	}
}

func TestRunnerUsesAPIOnlyWhenKeySet(t *testing.T) {
	freshConfig(t)
	var used string
	apiCall = func(key, model, system, input string) (Table, error) { used = "api:" + key; return noHeader, nil }
	claudeCall = func(model, system, input string) (Table, error) { used = "claude"; return noHeader, nil }
	t.Cleanup(func() { apiCall, claudeCall = runAPI, runClaude })
	autoRunner("haiku", "", "")
	if used != "claude" {
		t.Fatalf("no key: used %s", used)
	}
	fakeKey = "sk-1"
	autoRunner("haiku", "", "")
	if used != "api:sk-1" {
		t.Fatalf("with key: used %s", used)
	}
}

func TestModelFromFlagThenConfigThenDefault(t *testing.T) {
	freshConfig(t)
	clipboardWriter = func(string, string) error { return nil }
	t.Cleanup(func() { clipboardWriter = writeClipboard })
	var model string
	aiRunner = func(m, _, _ string) (Table, error) { model = m; return noHeader, nil }
	t.Cleanup(func() { aiRunner = autoRunner })
	cli := func(args ...string) {
		stdinPiped = func() bool { return true }
		run(append([]string{"md", "--stdout", "--ai", "clean"}, args...), strings.NewReader(md), io.Discard, io.Discard)
	}
	cli()
	if model != "haiku" {
		t.Fatalf("default model = %s", model)
	}
	runOK(t, "config", "set", "model", "sonnet")
	cli()
	if model != "sonnet" {
		t.Fatalf("config model = %s", model)
	}
	cli("--ai-model", "opus")
	if model != "opus" {
		t.Fatalf("flag model = %s", model)
	}
}

func TestEnhanceExpandsShortcut(t *testing.T) {
	system, _ := fakeAI(t, noHeader, nil)
	if _, err := Enhance(noHeader, "vi", "haiku", map[string]string{"vi": "translate to Vietnamese"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(*system, "translate to Vietnamese") || strings.Contains(*system, ": vi") {
		t.Fatalf("system = %s", *system)
	}
	if aiLabel("vi", map[string]string{"vi": "x"}) != "vi" || aiLabel("shorten", nil) != "custom" || aiLabel("clean", nil) != "clean" {
		t.Fatal("aiLabel")
	}
}

func TestAPIModelAliases(t *testing.T) {
	for in, want := range map[string]string{"haiku": "claude-haiku-5-5", "sonnet": "claude-sonnet-5-5", "opus": "claude-opus-5-5", "claude-x-1": "claude-x-1"} {
		if got := apiModel(in); got != want {
			t.Fatalf("apiModel(%s) = %s", in, got)
		}
	}
}

// TestRunAPIRequestAndResponse runs the real SDK call against a local server.
func TestRunAPIRequestAndResponse(t *testing.T) {
	var body map[string]any
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Api-Key")
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"msg_1","type":"message","role":"assistant","model":"claude-haiku-5-5",
			"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1},
			"content":[{"type":"text","text":"{\"header\":[\"Field\",\"Journal\"],\"rows\":[[\"Learning\",\"PSPI\"]]}"}]}`)
	}))
	defer srv.Close()
	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)
	got, err := runAPI("sk-test", "haiku", "SYSTEM", "INPUT")
	if err != nil {
		t.Fatal(err)
	}
	if want := (Table{Header: []string{"Field", "Journal"}, Rows: [][]string{{"Learning", "PSPI"}}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if gotKey != "sk-test" || body["model"] != "claude-haiku-5-5" {
		t.Fatalf("key=%q body=%v", gotKey, body)
	}
	format := body["output_config"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "json_schema" || format["schema"].(map[string]any)["required"] == nil {
		t.Fatalf("output_config = %v", body["output_config"])
	}
	if !strings.Contains(string(mustJSON(body["system"])), "SYSTEM") || !strings.Contains(string(mustJSON(body["messages"])), "INPUT") {
		t.Fatalf("body = %v", body)
	}
}

func TestRunAPIRefusalIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"msg_1","type":"message","role":"assistant","model":"m","stop_reason":"refusal",
			"usage":{"input_tokens":1,"output_tokens":1},"content":[]}`)
	}))
	defer srv.Close()
	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)
	if _, err := runAPI("sk", "haiku", "s", "i"); err == nil || !strings.Contains(err.Error(), "refusal") {
		t.Fatalf("err = %v", err)
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func decodeFilter(t *testing.T, query string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := alfredFilter(query, &out); err != nil {
		t.Fatal(err)
	}
	var v struct{ Items []map[string]any }
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v.Items
}

func TestAlfredCommands(t *testing.T) {
	freshConfig(t)
	clipboardReader = func() (string, error) { return md, nil }
	t.Cleanup(func() { clipboardReader = readClipboard })

	items := decodeFilter(t, "/set apikey sk-ant-secretsecret-WXYZ")
	if len(items) != 1 || items[0]["valid"] != true || items[0]["arg"] != "/set apikey sk-ant-secretsecret-WXYZ" ||
		strings.Contains(items[0]["title"].(string), "secretsecret") {
		t.Fatalf("items = %v", items)
	}
	if items := decodeFilter(t, "/set apikey"); len(items) != 1 || items[0]["valid"] != false {
		t.Fatalf("incomplete command is valid: %v", items)
	}
	if items := decodeFilter(t, "/"); len(items) < 4 || items[0]["valid"] != false {
		t.Fatalf("command list = %v", items)
	}

	// Enter on a command item runs it.
	var notice bytes.Buffer
	if err := alfredRun("/prompt add vi translate to Vietnamese", "", &notice); err != nil {
		t.Fatal(err)
	}
	if err := alfredRun("/set model sonnet", "", &notice); err != nil {
		t.Fatal(err)
	}
	if c, _ := loadConfig(); c.Prompts["vi"] != "translate to Vietnamese" || c.Model != "sonnet" {
		t.Fatalf("config = %+v", c)
	}
	if !strings.Contains(notice.String(), "vi") {
		t.Fatalf("notice = %s", notice.String())
	}

	// /config lists the settings.
	var titles []string
	for _, it := range decodeFilter(t, "/config") {
		titles = append(titles, it["title"].(string)+" "+it["subtitle"].(string))
	}
	if all := strings.Join(titles, "\n"); !strings.Contains(all, "sonnet") || !strings.Contains(all, "translate to Vietnamese") {
		t.Fatalf("/config = %s", all)
	}

	// A shortcut name runs the shortcut; the subtitle shows its text.
	items = decodeFilter(t, "slack vi")
	if len(items) != 1 || items[0]["variables"].(map[string]any)["ai"] != "vi" ||
		!strings.Contains(items[0]["subtitle"].(string), "translate to Vietnamese") {
		t.Fatalf("items = %v", items)
	}
	// With no query, the shortcuts follow the targets as Tab-complete hints.
	items = decodeFilter(t, "")
	last := items[len(items)-1]
	if len(items) != 6 || last["valid"] != false || last["autocomplete"] != "vi" {
		t.Fatalf("items = %v", items)
	}
}

func TestAlfredRunTargetUsesAIVariable(t *testing.T) {
	freshConfig(t)
	clipboardReader = func() (string, error) { return md, nil }
	var copied string
	clipboardWriter = func(plain, _ string) error { copied = plain; return nil }
	stdinPiped = func() bool { return false }
	t.Cleanup(func() { clipboardReader, clipboardWriter = readClipboard, writeClipboard })
	want := Table{Header: []string{"A", "B"}, Rows: noHeader.Rows}
	fakeAI(t, want, nil)
	var notice bytes.Buffer
	if err := alfredRun("md", "shorten each cell", &notice); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(copied, "| A ") || !strings.Contains(notice.String(), "AI step: custom") {
		t.Fatalf("copied=%q notice=%q", copied, notice.String())
	}
}
