package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Config is the settings file. The API key is not in it: it lives in the keychain.
type Config struct {
	Model   string            `json:"model,omitempty"`
	Prompts map[string]string `json:"prompts,omitempty"`
}

const defaultModel = "haiku"

// configPath is ~/Library/Application Support/pastefix/config.json on macOS.
// PASTEFIX_CONFIG overrides it (tests use this).
func configPath() (string, error) {
	if p := os.Getenv("PASTEFIX_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pastefix", "config.json"), nil
}

// loadConfig returns an empty config when the file does not exist, and an error
// when it exists but cannot be read, so a broken file is never silently ignored.
func loadConfig() (Config, error) {
	var c Config
	p, err := configPath()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("config file %s: %w", p, err)
	}
	return c, nil
}

func saveConfig(c Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// The API key is a generic password in the login keychain. Swapped in tests.
const keychainService, keychainAccount = "pastefix", "anthropic-api-key"

var (
	keychainGet = func() (string, error) {
		out, err := macCommand("security", "find-generic-password", "-s", keychainService, "-a", keychainAccount, "-w").Output()
		if err != nil {
			return "", nil // no item
		}
		return strings.TrimSpace(string(out)), nil
	}
	keychainSet = func(key string) error {
		out, err := macCommand("security", "add-generic-password", "-U", "-s", keychainService, "-a", keychainAccount, "-w", key).CombinedOutput()
		if err != nil {
			return fmt.Errorf("keychain: %v: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	keychainDelete = func() error {
		macCommand("security", "delete-generic-password", "-s", keychainService, "-a", keychainAccount).Run() //nolint:errcheck // a missing item is fine
		return nil
	}
)

func apiKey() string {
	k, _ := keychainGet()
	return k
}

func maskKey(k string) string {
	if len(k) <= 8 {
		return "…"
	}
	return k[:7] + "…" + k[len(k)-4:]
}

// promptNameError says why name cannot be a shortcut: it would be read as a
// target, a command or the built-in step.
func promptNameError(name string) error {
	switch {
	case name == "" || strings.ContainsAny(name, " \t\n"):
		return fmt.Errorf("shortcut name %q must be one word", name)
	case strings.HasPrefix(name, "/"):
		return fmt.Errorf("shortcut name %q cannot start with /", name)
	case slices.ContainsFunc(targets, func(tg target) bool { return tg.name == name }):
		return fmt.Errorf("%q is a target name", name)
	}
	if _, ok := aiSteps[name]; ok {
		return fmt.Errorf("%q is a built-in step", name)
	}
	return nil
}

// configCommand runs `config [set|unset KEY [VALUE]]` and `prompt [add|rm ...]`.
// The CLI and the Alfred / commands both come here. It returns the message to show.
func configCommand(words []string, out io.Writer) (string, error) {
	c, err := loadConfig()
	if err != nil {
		return "", err
	}
	switch {
	case len(words) == 1 && words[0] == "config", len(words) == 1 && words[0] == "prompt":
		for _, l := range configLines(c) {
			fmt.Fprintf(out, "%-9s %s\n", l.key, l.value)
		}
		return "", nil
	case len(words) == 4 && words[0] == "config" && words[1] == "set":
		key, value := words[2], words[3]
		switch key {
		case "apikey":
			if err := keychainSet(value); err != nil {
				return "", err
			}
			return "saved the API key " + maskKey(value) + " in the keychain; the AI step now calls the Anthropic API.", nil
		case "model":
			c.Model = value
			return "model is now " + value + ".", saveConfig(c)
		}
		return "", fmt.Errorf("unknown setting %q (apikey, model)", key)
	case len(words) == 3 && words[0] == "config" && words[1] == "unset":
		switch words[2] {
		case "apikey":
			return "removed the API key; the AI step uses your claude login.", keychainDelete()
		case "model":
			c.Model = ""
			return "model is now " + defaultModel + " (default).", saveConfig(c)
		}
		return "", fmt.Errorf("unknown setting %q (apikey, model)", words[2])
	case len(words) >= 4 && words[0] == "prompt" && words[1] == "add":
		name, text := words[2], strings.Join(words[3:], " ")
		if err := promptNameError(name); err != nil {
			return "", err
		}
		if c.Prompts == nil {
			c.Prompts = map[string]string{}
		}
		c.Prompts[name] = text
		return fmt.Sprintf("saved shortcut %s: %s", name, text), saveConfig(c)
	case len(words) == 3 && words[0] == "prompt" && words[1] == "rm":
		if _, ok := c.Prompts[words[2]]; !ok {
			return "", fmt.Errorf("no shortcut %q", words[2])
		}
		delete(c.Prompts, words[2])
		return "removed shortcut " + words[2] + ".", saveConfig(c)
	}
	return "", errors.New("usage: config | config set apikey|model VALUE | config unset apikey|model | " +
		"prompt | prompt add NAME TEXT... | prompt rm NAME")
}

type configLine struct{ key, value string }

// configLines describes every setting, for `pastefix config` and Alfred's /config.
func configLines(c Config) []configLine {
	p, _ := configPath()
	model := c.Model
	if model == "" {
		model = defaultModel + " (default)"
	}
	route := "not set; the AI step runs claude -p on your claude login"
	if k := apiKey(); k != "" {
		route = maskKey(k) + " (keychain); the AI step calls the Anthropic API"
	}
	claude := "not found"
	if b, err := claudeBin(); err == nil {
		claude = b
	}
	lines := []configLine{{"file", p}, {"apikey", route}, {"model", model}, {"claude", claude}}
	for _, name := range slices.Sorted(maps.Keys(aiSteps)) {
		lines = append(lines, configLine{"prompt", name + " (built in): " + aiSteps[name]})
	}
	for _, name := range slices.Sorted(maps.Keys(c.Prompts)) {
		lines = append(lines, configLine{"prompt", name + ": " + c.Prompts[name]})
	}
	return lines
}
