// Package cmd wires gh-board's cobra command tree.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/git"
	"github.com/SamuelBisberg/gh-board/pkg/tui"
)

// Version is injected at build time (see main.go / .github/workflows/release.yml).
var Version = "dev"

// NewRootCmd builds the `gh board` root command and all its subcommands.
func NewRootCmd() *cobra.Command {
	var all, dryRun bool
	root := &cobra.Command{
		Use:   "board",
		Short: "Keep a GitHub Project board in step with your work",
		Long: "gh-board keeps a GitHub Project Board in step with the work you do:\n" +
			"capture a Todo issue, start it, push, open it for review, and the card\n" +
			"moves by itself. With no subcommand it lists your Todo issues.",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			gh.DryRun, git.DryRun = dryRun, dryRun
		},
		RunE: func(cmd *cobra.Command, args []string) error { return runList(all) },
	}
	root.Flags().BoolVar(&all, "all", false, "list Todo issues of every repo")
	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print the planned gh/git calls and change nothing")
	root.AddCommand(newTodoCmd(), newNewCmd(), newPushCmd(), newOpenCmd(), newReviewCmd(),
		newTeamCmd(), newSyncCmd(), newInitCmd(), newConfigCmd())
	return root
}

// runList shows the interactive list of Todo issues; enter starts work.
func runList(all bool) error {
	s, err := openSession(false)
	if err != nil {
		return err
	}
	here := ""
	if git.IsRepo() {
		if r, err := gh.CurrentRepo(""); err == nil {
			here = r.NameWithOwner
		}
	}
	items, err := s.board.Items()
	if err != nil {
		return err
	}
	var todos []board.Item
	for _, it := range items {
		if it.Type == "Issue" && it.Status == board.Todo && (all || it.Repo == here) {
			todos = append(todos, it)
		}
	}
	if len(todos) == 0 {
		s.theme.Infof("No Todo issues")
		return nil
	}
	labels := make([]string, len(todos))
	for i, it := range todos {
		labels[i] = fmt.Sprintf("%s#%d  %s", it.Repo, it.Number, it.Title)
	}
	if !tui.IsInteractive() {
		for _, l := range labels {
			fmt.Println(l)
		}
		return nil
	}

	key, idx, err := tui.RunList("Todo", labels, []string{"enter", "o"}, "enter start work  o open in browser  esc quit")
	if err != nil {
		return err
	}
	it := todos[idx]
	switch key {
	case "o":
		_, err := gh.Read("issue", "view", it.URL, "--web")
		return err
	case "enter":
		if it.Repo != here {
			name := it.Repo[indexAfterSlash(it.Repo):]
			fmt.Printf("cd ~/work/%s && gh board new --issue %d\n", name, it.Number)
			return nil
		}
		return runNew("", "", "", it.Number)
	}
	return nil
}

func indexAfterSlash(s string) int {
	for i := range s {
		if s[i] == '/' {
			return i + 1
		}
	}
	return 0
}
