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
	var owner string
	var project int
	c := &cobra.Command{
		Use:   "init",
		Short: "Pick your Board, check it against board.yml and write the config",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runInit(owner, project) },
	}
	c.Flags().StringVar(&owner, "owner", "", "user or organisation that owns the Board (default: you)")
	c.Flags().IntVar(&project, "project", 0, "project number (skips the picker)")
	return c
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

func runInit(ownerFlag string, projectFlag int) error {
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

	if projectFlag > 0 {
		cfg.Board.Project = projectFlag
	} else if tui.IsInteractive() {
		var list struct {
			Projects []struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"projects"`
		}
		if err := gh.ReadJSON(&list, "project", "list", "--owner", cfg.Board.Owner, "--format", "json"); err != nil {
			return fmt.Errorf("listing projects: %w (try `gh auth refresh -s project -s read:org`)", err)
		}
		labels := []string{"Create a new board"}
		for _, p := range list.Projects {
			labels = append(labels, fmt.Sprintf("#%d %s", p.Number, p.Title))
		}
		idx, err := tui.PickOne("Which board?", labels)
		if err != nil {
			return err
		}
		if idx > 0 {
			cfg.Board.Project = list.Projects[idx-1].Number
		} else {
			cfg.Board.Project = 0
		}
	}

	if cfg.Board.Project == 0 {
		title := strings.ReplaceAll(def.Board.Title, "{{login}}", login)
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
