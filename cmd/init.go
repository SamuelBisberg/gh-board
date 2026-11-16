package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/tui"
)

func newInitCmd() *cobra.Command {
	var owner, title string
	c := &cobra.Command{
		Use:   "init",
		Short: "Name your Board, check it against board.yml and write the config",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runInit(owner, title) },
	}
	c.Flags().StringVar(&owner, "owner", "", "user or organisation that owns the Board (default: you)")
	c.Flags().StringVar(&title, "title", "", "Board name: reuse the Board with this exact title, or create it (skips the prompt)")
	return c
}

type project struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// listProjects returns the owner's projects.
func listProjects(owner string) ([]project, error) {
	out, err := gh.Read("project", "list", "--owner", owner, "--limit", "200", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w (try `gh auth refresh -s project -s read:org`)", err)
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	var list struct {
		Projects []project `json:"projects"`
	}
	if err := jsonUnmarshal(out, &list); err != nil {
		return nil, err
	}
	return list.Projects, nil
}

// withTitle returns the projects whose title is exactly title.
func withTitle(ps []project, title string) []project {
	var found []project
	for _, p := range ps {
		if p.Title == title {
			found = append(found, p)
		}
	}
	return found
}

func duplicateTitleError(title string, dups []project) error {
	nums := make([]string, len(dups))
	for i, d := range dups {
		nums[i] = fmt.Sprintf("#%d", d.Number)
	}
	return fmt.Errorf("several boards are titled %q (%s): rename all but one on GitHub", title, strings.Join(nums, ", "))
}

// resolveByTitle finds the Board titled title among ps without asking: 0 if
// none exists (create it), its number if one does.
func resolveByTitle(ps []project, title string) (int, error) {
	switch found := withTitle(ps, title); len(found) {
	case 0:
		return 0, nil
	case 1:
		return found[0].Number, nil
	default:
		return 0, duplicateTitleError(title, found)
	}
}

// promptBoard asks for the Board's name until it names a Board to reuse
// (its number) or a free title (0, create it). The configured Board is
// reused without asking again.
func promptBoard(ps []project, configured int, defaultTitle string) (number int, title string, err error) {
	prefill := defaultTitle
	for _, p := range ps {
		if p.Number == configured {
			prefill = p.Title
		}
	}
	for {
		answer, err := tui.Input("Board name?", prefill)
		if err != nil {
			return 0, "", err
		}
		title = strings.TrimSpace(answer)
		if title == "" {
			title = defaultTitle
		}
		prefill = title
		found := withTitle(ps, title)
		switch {
		case len(found) == 0:
			return 0, title, nil
		case len(found) == 1 && found[0].Number == configured:
			return configured, title, nil
		case len(found) == 1:
			ok, err := tui.Confirm(fmt.Sprintf("Use existing board #%d %q?", found[0].Number, title))
			if err != nil {
				return 0, "", err
			}
			if ok {
				return found[0].Number, title, nil
			}
		default:
			labels := make([]string, len(found))
			for i, f := range found {
				labels[i] = fmt.Sprintf("#%d %s", f.Number, f.Title)
			}
			idx, err := tui.PickOne("Several boards have that title. Which one?", labels)
			if err != nil {
				return 0, "", err
			}
			return found[idx].Number, title, nil
		}
	}
}

// chooseOwner returns whose board to use: --owner, else the configured
// owner, else (interactively) the active account or one of its
// organisations, else the active account.
func chooseOwner(flag, configured, login string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if configured != "" {
		return configured, nil
	}
	if tui.IsInteractive() {
		owners := []string{login}
		if out, err := gh.Read("api", "user/orgs", "--jq", ".[].login"); err == nil && out != "" {
			owners = append(owners, strings.Fields(out)...)
		}
		if len(owners) > 1 {
			idx, err := tui.PickOne("Whose board?", owners)
			if err != nil {
				return "", err
			}
			return owners[idx], nil
		}
	}
	return login, nil
}

func runInit(ownerFlag, titleFlag string) error {
	if err := checkAuth(); err != nil {
		return err
	}
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	def, err := board.LoadDefinition()
	if err != nil {
		return err
	}
	login, err := gh.Login()
	if err != nil {
		return fmt.Errorf("finding the active account: %w", err)
	}
	prevOwner := cfg.Board.Owner
	owner, err := chooseOwner(ownerFlag, cfg.Board.Owner, login)
	if err != nil {
		return err
	}
	if owner != prevOwner {
		cfg.Board.Project = 0 // a project number only means something under its owner
	}
	cfg.Board.Owner = owner

	defaultTitle := strings.ReplaceAll(def.Board.Title, "{{login}}", login)
	title := defaultTitle
	titleFlag = strings.TrimSpace(titleFlag)
	switch {
	case titleFlag == "" && cfg.Board.Project > 0 && !tui.IsInteractive():
		// scripted re-run: keep the configured Board
	default:
		ps, err := listProjects(cfg.Board.Owner)
		if err != nil {
			return err
		}
		switch {
		case titleFlag != "":
			title = titleFlag
			cfg.Board.Project, err = resolveByTitle(ps, title)
		case tui.IsInteractive():
			cfg.Board.Project, title, err = promptBoard(ps, cfg.Board.Project, defaultTitle)
		default:
			cfg.Board.Project, err = resolveByTitle(ps, title)
		}
		if err != nil {
			return err
		}
	}

	if cfg.Board.Project == 0 {
		var created struct {
			Number int `json:"number"`
		}
		out, err := gh.Write("project", "create", "--owner", cfg.Board.Owner, "--title", title, "--format", "json")
		if err != nil {
			return fmt.Errorf("creating the board: %w", err)
		}
		if out != "" {
			if err := jsonUnmarshal(out, &created); err != nil {
				return err
			}
		}
		cfg.Board.Project = created.Number
		theme.Successf("Created board %q (#%d)", title, created.Number)
	}

	live, err := board.ReadLive(cfg.Board.Owner, cfg.Board.Project)
	if err != nil {
		return fmt.Errorf("reading board %s #%d: %w (try `gh auth refresh -s project -s read:org`)", cfg.Board.Owner, cfg.Board.Project, err)
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	theme.Successf("Using board %s #%d", cfg.Board.Owner, cfg.Board.Project)

	// Everything GitHub's API allows is done for you: the Status field and
	// the board view are created or corrected to match board.yml.
	done, err := def.Setup(cfg.Board.Owner, cfg.Board.Project, live)
	for _, d := range done {
		theme.Successf("%s", d)
	}
	if err != nil {
		return err
	}
	if len(done) == 0 {
		theme.Successf("Status field and board view already match board.yml")
	}

	// Workflows are the one thing GitHub exposes read-only, so they are
	// advice: sync re-derives every Status regardless.
	ownerType, _ := gh.Read("api", "users/"+cfg.Board.Owner, "--jq", ".type")
	url := board.SettingsURL(ownerType, cfg.Board.Owner, cfg.Board.Project)
	if advice := def.WorkflowAdvice(live, url); len(advice) > 0 {
		theme.Warningf("GitHub has no API for project workflows. Optional, so cards move the instant a review happens (`gh board sync` fixes the rest):")
		for _, a := range advice {
			fmt.Printf("  - %s\n", a)
		}
	} else {
		theme.Successf("The board's status workflows are set as expected")
	}
	return nil
}
