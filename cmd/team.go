package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/config"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/tui"
)

func newTeamCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "team",
		Short: "Manage the teammates you can request as reviewers",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runTeamInteractive() },
	}
	c.AddCommand(
		&cobra.Command{
			Use: "add <login> [alias]", Short: "Save a teammate", Args: cobra.RangeArgs(1, 2),
			RunE: func(cmd *cobra.Command, args []string) error {
				alias := ""
				if len(args) > 1 {
					alias = args[1]
				}
				return runTeamAdd(args[0], alias)
			},
		},
		&cobra.Command{
			Use: "remove <name>", Short: "Remove a teammate by alias or login", Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error { return runTeamRemove(args[0]) },
		},
		&cobra.Command{
			Use: "list", Short: "List teammates", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error { return runTeamList() },
		},
		&cobra.Command{
			Use: "default [name ...]", Short: "Set (or print) the default reviewers",
			RunE: func(cmd *cobra.Command, args []string) error { return runTeamDefault(splitNames(args)) },
		},
	)
	return c
}

func isDefault(cfg *config.Config, alias string) bool {
	for _, d := range cfg.Reviewers.Default {
		if strings.EqualFold(d, alias) {
			return true
		}
	}
	return false
}

func runTeamAdd(login, alias string) error {
	if err := checkAuth(); err != nil {
		return err
	}
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	canonical, err := gh.Read("api", "users/"+login, "--jq", ".login")
	if err != nil || canonical == "" {
		return fmt.Errorf("no GitHub user %q", login)
	}
	if _, err := gh.Read("api", "orgs/"+cfg.Board.Org+"/members/"+canonical, "--silent"); err != nil {
		return fmt.Errorf("%s is not a member of %s", canonical, cfg.Board.Org)
	}
	if alias == "" {
		alias = canonical
	}
	if _, taken := cfg.FindTeammate(alias); taken {
		return fmt.Errorf("alias %q is already in use", alias)
	}
	cfg.Team = append(cfg.Team, config.Teammate{Alias: alias, Login: canonical})
	if err := cfg.Save(); err != nil {
		return err
	}
	theme.Successf("Saved %s as %q", canonical, alias)
	return nil
}

func runTeamRemove(name string) error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	t, ok := cfg.FindTeammate(name)
	if !ok {
		return fmt.Errorf("no teammate %q", name)
	}
	removeTeammate(cfg, t)
	if err := cfg.Save(); err != nil {
		return err
	}
	theme.Successf("Removed %s", t.Alias)
	return nil
}

func removeTeammate(cfg *config.Config, t config.Teammate) {
	var team []config.Teammate
	for _, x := range cfg.Team {
		if x != t {
			team = append(team, x)
		}
	}
	cfg.Team = team
	var defs []string
	for _, d := range cfg.Reviewers.Default {
		if !strings.EqualFold(d, t.Alias) && !strings.EqualFold(d, t.Login) {
			defs = append(defs, d)
		}
	}
	cfg.Reviewers.Default = defs
}

func runTeamList() error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	if len(cfg.Team) == 0 {
		theme.Infof("No teammates yet: gh board team add <login> [alias]")
		return nil
	}
	for _, t := range cfg.Team {
		mark := ""
		if isDefault(cfg, t.Alias) {
			mark = "  (default)"
		}
		fmt.Printf("%-16s %s%s\n", t.Alias, t.Login, mark)
	}
	return nil
}

func runTeamDefault(names []string) error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Println(strings.Join(cfg.Reviewers.Default, " "))
		return nil
	}
	var aliases []string
	for _, n := range names {
		t, ok := cfg.FindTeammate(n)
		if !ok {
			return fmt.Errorf("no teammate %q", n)
		}
		aliases = append(aliases, t.Alias)
	}
	cfg.Reviewers.Default = aliases
	if err := cfg.Save(); err != nil {
		return err
	}
	theme.Successf("Default reviewers: %s", strings.Join(aliases, ", "))
	return nil
}

func runTeamInteractive() error {
	for {
		cfg, theme, err := loadTheme()
		if err != nil {
			return err
		}
		if len(cfg.Team) == 0 {
			theme.Infof("No teammates yet: gh board team add <login> [alias]")
			return nil
		}
		if !tui.IsInteractive() {
			return runTeamList()
		}
		items := make([]string, len(cfg.Team))
		for i, t := range cfg.Team {
			star := " "
			if isDefault(cfg, t.Alias) {
				star = "*"
			}
			items[i] = fmt.Sprintf("%s %-16s %s", star, t.Alias, t.Login)
		}
		key, idx, err := tui.RunList("Teammates", items, []string{"d", "*"}, "d remove  * toggle default  esc quit")
		if err != nil {
			return err
		}
		t := cfg.Team[idx]
		switch key {
		case "d":
			if ok, _ := tui.Confirm("Remove " + t.Alias + "?"); !ok {
				continue
			}
			removeTeammate(cfg, t)
		case "*":
			if isDefault(cfg, t.Alias) {
				var defs []string
				for _, d := range cfg.Reviewers.Default {
					if !strings.EqualFold(d, t.Alias) {
						defs = append(defs, d)
					}
				}
				cfg.Reviewers.Default = defs
			} else {
				cfg.Reviewers.Default = append(cfg.Reviewers.Default, t.Alias)
			}
		default:
			return nil
		}
		if err := cfg.Save(); err != nil {
			return err
		}
	}
}
