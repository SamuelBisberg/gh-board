package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/config"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/git"
	"github.com/SamuelBisberg/gh-board/pkg/tmpl"
	"github.com/SamuelBisberg/gh-board/pkg/tui"
)

// loadTheme loads the persisted config and builds the color theme - the
// first thing every subcommand needs before it can print anything.
func loadTheme() (*config.Config, *tui.Theme, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}
	switch cfg.UI.Color {
	case "never":
		os.Setenv("NO_COLOR", "1")
	case "always":
		os.Setenv("CLICOLOR_FORCE", "1")
	}
	return cfg, tui.New(), nil
}

// session is everything a board command needs: config, theme, the
// logged-in user and the resolved Board.
type session struct {
	cfg   *config.Config
	theme *tui.Theme
	login string
	board *board.Board
}

// openSession checks auth, loads config and resolves the user and Board.
// needGit also requires the current directory to be a git repository.
func openSession(needGit bool) (*session, error) {
	if needGit && !git.IsRepo() {
		return nil, fmt.Errorf("not a git repository")
	}
	if err := checkAuth(); err != nil {
		return nil, err
	}
	cfg, theme, err := loadTheme()
	if err != nil {
		return nil, err
	}
	login, err := gh.Login()
	if err != nil {
		return nil, fmt.Errorf("finding the logged-in user: %w", err)
	}
	b, err := board.Open(cfg.Board.Org, cfg.Board.Project)
	if err != nil {
		return nil, err
	}
	return &session{cfg: cfg, theme: theme, login: login, board: b}, nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug turns an issue title into a branch-safe slug.
func slug(title string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if len(s) > 50 {
		s = strings.Trim(s[:50], "-")
	}
	return s
}

// prTitle derives a PR title from a branch name: dashes become spaces.
func prTitle(branch string) string {
	return strings.ReplaceAll(branch, "-", " ")
}

// prBody renders the PR template (repo's own, else user override, else
// embedded) and appends `Closes #N` when starting from an issue.
func prBody(repo, branch, base, title string, issue int) string {
	root, _ := git.RepoRoot()
	dir, _ := config.TemplatesDir()
	text, _ := tmpl.Resolve(tmpl.PR, root, dir)
	v := tmpl.Vars{Title: title, Branch: branch, Base: base, Repo: repo}
	if issue > 0 {
		v.Issue = fmt.Sprint(issue)
	}
	body := tmpl.Render(text, v)
	if issue > 0 {
		body = tmpl.WithCloses(body, issue)
	}
	return body
}

// trackPR sets a PR's Card from its current state. PRs closed without
// merging are left to the Board's own workflow.
func (s *session) trackPR(pr *gh.PR) error {
	status, ok := board.Derive(pr.IsDraft, pr.State, pr.ReviewDecision)
	if !ok {
		s.theme.Infof("%s is closed without merging; leaving its card to the board's workflow", pr.URL)
		return nil
	}
	if err := s.board.Track(pr.URL, status); err != nil {
		return err
	}
	s.theme.Successf("%s -> %s", pr.URL, status)
	return nil
}

// defaultBranch returns the current repo's default branch.
func defaultBranch() (repo, branch string, err error) {
	r, err := gh.CurrentRepo("")
	if err != nil {
		return "", "", fmt.Errorf("finding the repository: %w", err)
	}
	return r.NameWithOwner, r.DefaultBranch.Name, nil
}

// resolveReviewers picks reviewer logins: -r names, else --no-reviewers
// (nobody), else reviewers.default, else the interactive picker. The PR
// author is never requested.
func (s *session) resolveReviewers(names []string, none bool, author string) ([]string, error) {
	if none {
		return nil, nil
	}
	if len(names) == 0 {
		names = s.cfg.Reviewers.Default
	}
	if len(names) == 0 && tui.IsInteractive() && len(s.cfg.Team) > 0 {
		aliases := make([]string, len(s.cfg.Team))
		for i, t := range s.cfg.Team {
			aliases[i] = t.Alias
		}
		picked, err := tui.PickMany("Request review from", aliases)
		if err != nil {
			return nil, err
		}
		names = picked
	}
	seen := map[string]bool{strings.ToLower(author): true}
	var logins []string
	for _, n := range names {
		l := s.cfg.ResolveLogin(strings.TrimSpace(n))
		if l == "" || seen[strings.ToLower(l)] {
			continue
		}
		seen[strings.ToLower(l)] = true
		logins = append(logins, l)
	}
	return logins, nil
}

func splitNames(csv []string) []string {
	var out []string
	for _, c := range csv {
		for _, n := range strings.Split(c, ",") {
			if n = strings.TrimSpace(n); n != "" {
				out = append(out, n)
			}
		}
	}
	return out
}

func jsonUnmarshal(s string, v any) error {
	if err := json.Unmarshal([]byte(s), v); err != nil {
		return fmt.Errorf("decoding gh output: %w", err)
	}
	return nil
}
