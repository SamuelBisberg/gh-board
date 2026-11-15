package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/git"
)

func newNewCmd() *cobra.Command {
	var title string
	var issue int
	c := &cobra.Command{
		Use:   "new [branch] [base]",
		Short: "Start work: branch, draft PR, In Progress",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch, base := "", ""
			if len(args) > 0 {
				branch = args[0]
			}
			if len(args) > 1 {
				base = args[1]
			}
			return runNew(branch, base, title, issue)
		},
	}
	c.Flags().StringVarP(&title, "title", "t", "", "PR title (default: the branch name)")
	c.Flags().IntVar(&issue, "issue", 0, "start work on this issue")
	return c
}

func runNew(branch, base, title string, issueNum int) error {
	s, err := openSession(true)
	if err != nil {
		return err
	}
	repo, defBranch, err := defaultBranch()
	if err != nil {
		return err
	}
	if base == "" {
		base = defBranch
	}

	var issue *gh.Issue
	if issueNum > 0 {
		if issue, err = gh.IssueView("", fmt.Sprint(issueNum)); err != nil {
			return fmt.Errorf("reading issue #%d: %w", issueNum, err)
		}
		if branch == "" {
			branch = s.cfg.Branch.Prefix + "/" + slug(issue.Title)
		}
	}
	if branch == "" {
		return fmt.Errorf("a branch name or --issue is required")
	}
	if title == "" {
		title = prTitle(branch)
	}

	if err := git.Fetch(); err != nil {
		return fmt.Errorf("fetching origin: %w", err)
	}
	start := "origin/" + base
	if issue != nil {
		if err := gh.IssueDevelop(issue.Number, branch, base); err != nil {
			return fmt.Errorf("linking branch to issue: %w", err)
		}
		if err := git.Fetch(); err != nil {
			return fmt.Errorf("fetching origin: %w", err)
		}
		start = "origin/" + branch
	}
	if err := git.CreateBranchFrom(branch, start); err != nil {
		return err
	}
	if err := git.EmptyCommit("chore: start " + branch); err != nil {
		return err
	}
	if err := git.Push(); err != nil {
		return err
	}

	num := 0
	if issue != nil {
		num = issue.Number
	}
	url, err := gh.PRCreate(gh.PRCreateOpts{
		Title: title, Body: prBody(repo, branch, base, title, num), Base: base, Head: branch, Draft: true,
	})
	if err != nil {
		return fmt.Errorf("creating pull request: %w", err)
	}
	if err := s.board.Track(url, board.InProgress); err != nil {
		return err
	}
	s.theme.Successf("%s -> %s", url, board.InProgress)

	if issue != nil {
		items, err := s.board.Items()
		if err != nil {
			return err
		}
		if card, ok := board.Find(items, issue.URL); ok {
			if err := s.board.Remove(card.ID); err != nil {
				return err
			}
			s.theme.Infof("Removed the card for issue #%d (the PR's card replaces it)", issue.Number)
		}
	}
	return nil
}

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "push [git push args]",
		Short:              "Push and keep the card in step (opens a draft PR if none)",
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(args)
		},
	}
}

func runPush(pushArgs []string) error {
	s, err := openSession(true)
	if err != nil {
		return err
	}
	branch, err := git.CurrentBranch()
	if err != nil {
		return fmt.Errorf("determining current branch: %w", err)
	}
	repo, defBranch, err := defaultBranch()
	if err != nil {
		return err
	}
	if branch == defBranch {
		return fmt.Errorf("refusing to push the default branch %q", branch)
	}
	if err := git.Push(pushArgs...); err != nil {
		return err
	}

	pr, err := gh.PRView(branch)
	if err != nil {
		return err
	}
	if pr == nil {
		title := prTitle(branch)
		url, err := gh.PRCreate(gh.PRCreateOpts{
			Title: title, Body: prBody(repo, branch, defBranch, title, 0), Base: defBranch, Head: branch, Draft: true,
		})
		if err != nil {
			return fmt.Errorf("creating pull request: %w", err)
		}
		if err := s.board.Track(url, board.InProgress); err != nil {
			return err
		}
		s.theme.Successf("%s -> %s", url, board.InProgress)
		return nil
	}
	return s.trackPR(pr)
}
