// Package tmpl produces the body text for new issues and pull requests.
package tmpl

import (
	"embed"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed defaults/*.md
var defaults embed.FS

// Kind selects which template is used.
type Kind string

const (
	Issue Kind = "issue"
	PR    Kind = "pr"
)

// Source says where a template came from.
type Source string

const (
	SourceRepo     Source = "repo"
	SourceOverride Source = "override"
	SourceEmbedded Source = "embedded"
)

// Vars are the placeholder values.
type Vars struct {
	Title, Branch, Base, Repo, Issue string
}

var repoPaths = map[Kind][]string{
	PR: {".github/PULL_REQUEST_TEMPLATE.md", ".github/pull_request_template.md",
		"docs/pull_request_template.md", "pull_request_template.md"},
	Issue: {".github/ISSUE_TEMPLATE.md"},
}

// Default returns the embedded template for kind.
func Default(kind Kind) string {
	b, _ := defaults.ReadFile("defaults/" + string(kind) + ".md")
	return string(b)
}

// Resolve finds the template text: the repo's own template (looked up under
// repoRoot, skipped when empty), else the user's override in overrideDir
// (skipped when empty), else the embedded default.
func Resolve(kind Kind, repoRoot, overrideDir string) (string, Source) {
	if repoRoot != "" {
		for _, p := range repoPaths[kind] {
			if b, err := os.ReadFile(filepath.Join(repoRoot, p)); err == nil {
				return string(b), SourceRepo
			}
		}
	}
	if overrideDir != "" {
		if b, err := os.ReadFile(filepath.Join(overrideDir, string(kind)+".md")); err == nil {
			return string(b), SourceOverride
		}
	}
	return Default(kind), SourceEmbedded
}

// Render replaces known placeholders; unknown ones are left as written.
func Render(text string, v Vars) string {
	return strings.NewReplacer(
		"{{title}}", v.Title, "{{branch}}", v.Branch, "{{base}}", v.Base,
		"{{repo}}", v.Repo, "{{issue}}", v.Issue,
	).Replace(text)
}

// WithCloses appends a `Closes #N` line unless the body already has it.
func WithCloses(body string, issue int) string {
	line := "Closes #" + strconv.Itoa(issue)
	if strings.Contains(body, line) {
		return body
	}
	return strings.TrimRight(body, "\n") + "\n\n" + line + "\n"
}

// WriteDefaults writes the embedded templates into dir, without
// overwriting files that already exist. It returns the files written.
func WriteDefaults(dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var written []string
	for _, k := range []Kind{Issue, PR} {
		path := filepath.Join(dir, string(k)+".md")
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(Default(k)), 0o644); err != nil {
			return written, err
		}
		written = append(written, path)
	}
	return written, nil
}
