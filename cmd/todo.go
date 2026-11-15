package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/config"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/git"
	"github.com/SamuelBisberg/gh-board/pkg/tmpl"
)

func newTodoCmd() *cobra.Command {
	var body, repo string
	c := &cobra.Command{
		Use:   "todo <title>",
		Short: "Capture work as an issue on the Board (Todo)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTodo(args[0], body, repo)
		},
	}
	c.Flags().StringVarP(&body, "body", "b", "", "issue body (skips templates)")
	c.Flags().StringVarP(&repo, "repo", "R", "", "repository (owner/repo)")
	return c
}

func runTodo(title, body, repoFlag string) error {
	s, err := openSession(false)
	if err != nil {
		return err
	}

	repo, inRepo := repoFlag, false
	if repo == "" && git.IsRepo() {
		if r, err := gh.CurrentRepo(""); err == nil {
			repo, inRepo = r.NameWithOwner, true
		}
	}
	if repo == "" {
		repo = s.cfg.Todo.Repo
	}
	if repo == "" {
		return fmt.Errorf("no repository: run inside a repo, set todo.repo, or pass -R owner/repo")
	}

	if body == "" {
		root := ""
		if inRepo {
			root, _ = git.RepoRoot()
		}
		dir, _ := config.TemplatesDir()
		text, _ := tmpl.Resolve(tmpl.Issue, root, dir)
		body = tmpl.Render(text, tmpl.Vars{Title: title, Repo: repo})
	}

	url, err := gh.IssueCreate(repo, title, body)
	if err != nil {
		return fmt.Errorf("creating issue: %w", err)
	}
	if err := s.board.Track(url, board.Todo); err != nil {
		return err
	}
	fmt.Println(url)
	return nil
}
