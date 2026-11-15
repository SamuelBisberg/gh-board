package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/git"
)

func newOpenCmd() *cobra.Command {
	var title string
	var reviewers []string
	var none bool
	var issue int
	c := &cobra.Command{
		Use:   "open",
		Short: "Open the PR for review (Needs Review) and request reviewers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpen(title, splitNames(reviewers), none, issue)
		},
	}
	c.Flags().StringVarP(&title, "title", "t", "", "PR title (default: the branch name)")
	c.Flags().StringSliceVarP(&reviewers, "reviewers", "r", nil, "reviewers (teammate aliases or logins)")
	c.Flags().BoolVar(&none, "no-reviewers", false, "request nobody")
	c.Flags().IntVar(&issue, "issue", 0, "link this issue when a new PR is opened")
	return c
}

func runOpen(title string, names []string, none bool, issueNum int) error {
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
		return fmt.Errorf("refusing to open a PR from the default branch %q", branch)
	}

	issue, err := issueNumber(issueNum)
	if err != nil {
		return err
	}
	pr, err := gh.PRView(branch)
	if err != nil {
		return err
	}
	if pr != nil && pr.State != "OPEN" {
		return fmt.Errorf("the PR for %s is %s", branch, pr.State)
	}
	base := defBranch
	if pr != nil {
		base = pr.BaseRefName
	}

	if err := git.Fetch(); err != nil {
		return fmt.Errorf("fetching origin: %w", err)
	}
	ahead, err := git.CommitsAhead("origin/" + base)
	if err != nil {
		return fmt.Errorf("counting commits ahead of %s: %w", base, err)
	}
	if ahead == 0 {
		return fmt.Errorf("no commits ahead of %s; nothing to open", base)
	}

	// Reviewers: always for a PR becoming ready; only on request for one
	// that already was, so a plain `open` never re-notifies anybody.
	var reviewers []string
	if pr == nil || pr.IsDraft || len(names) > 0 {
		if reviewers, err = s.resolveReviewers(names, none, s.login); err != nil {
			return err
		}
	}

	if err := git.Push(); err != nil {
		return err
	}

	switch {
	case pr == nil:
		if title == "" {
			title = prTitle(branch)
		}
		url, err := gh.PRCreate(gh.PRCreateOpts{
			Title: title, Body: prBody(repo, branch, base, title, issueNum), Base: base, Head: branch, Reviewers: reviewers,
		})
		if err != nil {
			return fmt.Errorf("creating pull request: %w", err)
		}
		if err := s.board.Track(url, board.NeedsReview); err != nil {
			return err
		}
		s.theme.Successf("%s -> %s", url, board.NeedsReview)
		if issue != nil {
			return s.dropIssueCard(issue)
		}
		return nil
	case pr.IsDraft:
		if err := gh.PRReady(branch); err != nil {
			return fmt.Errorf("marking the PR ready: %w", err)
		}
		if err := gh.PRAddReviewers(branch, reviewers); err != nil {
			return fmt.Errorf("requesting reviewers: %w", err)
		}
		pr.IsDraft = false
	default:
		if err := gh.PRAddReviewers(branch, reviewers); err != nil {
			return fmt.Errorf("requesting reviewers: %w", err)
		}
	}
	if issue != nil {
		s.theme.Warningf("--issue only applies when a new PR is opened; the existing PR's body is unchanged")
	}
	return s.trackPR(pr)
}

func newReviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "review [name ...]",
		Short: "Request reviewers on the current branch's open PR",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview(splitNames(args))
		},
	}
}

func runReview(names []string) error {
	s, err := openSession(true)
	if err != nil {
		return err
	}
	branch, err := git.CurrentBranch()
	if err != nil {
		return fmt.Errorf("determining current branch: %w", err)
	}
	pr, err := gh.PRView(branch)
	if err != nil {
		return err
	}
	switch {
	case pr == nil:
		return fmt.Errorf("no pull request for %s", branch)
	case pr.State != "OPEN":
		return fmt.Errorf("the PR for %s is %s", branch, pr.State)
	case pr.IsDraft:
		return fmt.Errorf("the PR is a draft: run `gh board open` first")
	}

	logins, err := s.resolveReviewers(names, false, pr.Author.Login)
	if err != nil {
		return err
	}
	pending := map[string]bool{}
	for _, r := range pr.ReviewRequests {
		pending[r.Login] = true
	}
	var fresh []string
	for _, l := range logins {
		if pending[l] {
			s.theme.Infof("%s already has a pending request", l)
			continue
		}
		fresh = append(fresh, l)
	}
	if len(fresh) == 0 {
		s.theme.Infof("No new reviewers to request")
		return nil
	}
	if err := gh.PRAddReviewers(branch, fresh); err != nil {
		return fmt.Errorf("requesting reviewers: %w", err)
	}
	s.theme.Successf("Requested review from %v", fresh)
	return nil
}
