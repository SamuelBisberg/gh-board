// Package config manages gh-board's persistent user settings.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Teammate is a saved GitHub user with a short alias.
type Teammate struct {
	Alias string `json:"alias"`
	Login string `json:"login"`
}

type BoardConfig struct {
	Owner   string `json:"owner"`
	Project int    `json:"project"`
}

type TodoConfig struct {
	Repo string `json:"repo"`
}

type BranchConfig struct {
	Prefix string `json:"prefix"`
}

type ReviewersConfig struct {
	Default []string `json:"default"`
}

type UIConfig struct {
	Color string `json:"color"`
}

// Config holds all persisted gh-board settings.
type Config struct {
	Board     BoardConfig     `json:"board"`
	Todo      TodoConfig      `json:"todo"`
	Branch    BranchConfig    `json:"branch"`
	Team      []Teammate      `json:"team"`
	Reviewers ReviewersConfig `json:"reviewers"`
	UI        UIConfig        `json:"ui"`
}

func defaults() *Config {
	return &Config{
		Branch: BranchConfig{Prefix: "task"},
		UI:     UIConfig{Color: "auto"},
	}
}

// Dir returns gh-board's config directory root: <user config dir>/gh.
func Dir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gh"), nil
}

// Path returns the on-disk location of the config file (gh/board.json).
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "board.json"), nil
}

// TemplatesDir returns where user template overrides live.
func TemplatesDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "board", "templates"), nil
}

// Load reads the config file, returning defaults if none exists yet.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaults(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	cfg := defaults()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config at %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the config file, creating its parent directory if needed.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("writing config at %s: %w", path, err)
	}
	return nil
}

type settingField struct {
	key string
	get func(*Config) string
	set func(*Config, string) error
}

var fields = []settingField{
	{"board.owner",
		func(c *Config) string { return c.Board.Owner },
		func(c *Config, v string) error { c.Board.Owner = v; return nil }},
	{"board.project",
		func(c *Config) string { return strconv.Itoa(c.Board.Project) },
		func(c *Config, v string) error {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return fmt.Errorf("board.project must be a positive number, got %q", v)
			}
			c.Board.Project = n
			return nil
		}},
	{"todo.repo",
		func(c *Config) string { return c.Todo.Repo },
		func(c *Config, v string) error {
			if v != "" && strings.Count(v, "/") != 1 {
				return fmt.Errorf("todo.repo must look like owner/repo, got %q", v)
			}
			c.Todo.Repo = v
			return nil
		}},
	{"branch.prefix",
		func(c *Config) string { return c.Branch.Prefix },
		func(c *Config, v string) error { c.Branch.Prefix = strings.Trim(v, "/"); return nil }},
	{"reviewers.default",
		func(c *Config) string { return strings.Join(c.Reviewers.Default, ",") },
		func(c *Config, v string) error {
			c.Reviewers.Default = nil
			for _, name := range strings.Split(v, ",") {
				if name = strings.TrimSpace(name); name != "" {
					c.Reviewers.Default = append(c.Reviewers.Default, name)
				}
			}
			return nil
		}},
	{"ui.color",
		func(c *Config) string { return c.UI.Color },
		func(c *Config, v string) error {
			switch v {
			case "auto", "always", "never":
				c.UI.Color = v
				return nil
			}
			return fmt.Errorf("invalid ui.color %q (want one of: auto, always, never)", v)
		}},
}

// Keys lists the settable config keys in display order. The team is edited
// with `gh board team`, not here.
func Keys() []string {
	keys := make([]string, len(fields))
	for i, f := range fields {
		keys[i] = f.key
	}
	return keys
}

// Get reads a setting by key.
func (c *Config) Get(key string) (string, error) {
	for _, f := range fields {
		if f.key == key {
			return f.get(c), nil
		}
	}
	if key == "team" {
		parts := make([]string, len(c.Team))
		for i, t := range c.Team {
			parts[i] = t.Alias + "=" + t.Login
		}
		return strings.Join(parts, ","), nil
	}
	return "", fmt.Errorf("unknown config key %q", key)
}

// Set writes a setting by key.
func (c *Config) Set(key, value string) error {
	for _, f := range fields {
		if f.key == key {
			return f.set(c, value)
		}
	}
	if key == "team" {
		return fmt.Errorf("team is edited with `gh board team`")
	}
	return fmt.Errorf("unknown config key %q", key)
}

// FindTeammate resolves an alias (case-insensitive) or login to a teammate.
func (c *Config) FindTeammate(name string) (Teammate, bool) {
	for _, t := range c.Team {
		if strings.EqualFold(t.Alias, name) || strings.EqualFold(t.Login, name) {
			return t, true
		}
	}
	return Teammate{}, false
}

// ResolveLogin turns a teammate alias into its login; unknown names are
// treated as plain GitHub logins.
func (c *Config) ResolveLogin(name string) string {
	if t, ok := c.FindTeammate(name); ok {
		return t.Login
	}
	return name
}
