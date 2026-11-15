package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/git"
)

func newSyncCmd() *cobra.Command {
	var current bool
	c := &cobra.Command{
		Use:   "sync",
		Short: "Re-derive every card's Status from its PR",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runSync(current) },
	}
	c.Flags().BoolVar(&current, "current", false, "only the current branch in step 1")
	return c
}

type syncRow struct{ what, old, result string }

// prCache looks each PR up once per run.
type prCache map[string]*gh.PR

func (c prCache) get(ref string) (*gh.PR, error) {
	if pr, ok := c[ref]; ok {
		return pr, nil
	}
	pr, err := gh.PRView(ref)
	if err != nil {
		return nil, err
	}
	c[ref] = pr
	return pr, nil
}

func runSync(current bool) error {
	s, err := openSession(true)
	if err != nil {
		return err
	}
	var rows []syncRow
	var skipped []string

	// 1. Local branches of the current repo that have a PR of mine.
	branches := []string{}
	if current {
		b, err := git.CurrentBranch()
		if err != nil {
			return fmt.Errorf("determining current branch: %w", err)
		}
		branches = append(branches, b)
	} else if branches, err = git.Branches(); err != nil {
		return err
	}
	before, err := s.board.Items()
	if err != nil {
		return err
	}
	for _, br := range branches {
		pr, err := gh.PRView(br)
		if err != nil {
			return err
		}
		if pr == nil {
			skipped = append(skipped, br)
			continue
		}
		if pr.Author.Login != s.login || pr.State != "OPEN" {
			continue
		}
		want, _ := board.Derive(pr.IsDraft, pr.State, pr.ReviewDecision)
		old := board.Status("-")
		if it, ok := board.Find(before, pr.URL); ok {
			if it.Status == want || it.Status == board.Closed {
				continue
			}
			old = it.Status
		}
		if err := s.board.Track(pr.URL, want); err != nil {
			return err
		}
		rows = append(rows, syncRow{pr.URL, string(old), string(want)})
	}

	// 2. Refresh every PR of mine on the Board, in any repo.
	items, err := s.board.Items()
	if err != nil {
		return err
	}
	prs := prCache{}
	for _, it := range items {
		if it.Type != "PullRequest" || it.Status == board.Closed {
			continue
		}
		pr, err := prs.get(it.URL)
		if err != nil {
			return err
		}
		if pr == nil || pr.Author.Login != s.login {
			continue
		}
		want, ok := board.Derive(pr.IsDraft, pr.State, pr.ReviewDecision)
		if !ok || want == it.Status {
			continue
		}
		if err := s.board.SetStatus(it.ID, want); err != nil {
			return err
		}
		rows = append(rows, syncRow{it.URL, string(it.Status), string(want)})
	}

	// 3. Reconcile issue cards against their PRs.
	for _, it := range items {
		switch {
		case it.Type == "Issue" && it.Status != board.Closed:
			issue, err := gh.IssueView("", it.URL)
			if err != nil {
				return err
			}
			if issue.Author.Login != s.login {
				continue
			}
			for _, ref := range issue.ClosedBy {
				pr, err := prs.get(ref.URL)
				if err != nil {
					return err
				}
				if pr != nil && pr.Author.Login == s.login && pr.State != "CLOSED" {
					if err := s.board.Remove(it.ID); err != nil {
						return err
					}
					rows = append(rows, syncRow{it.URL, string(it.Status), "card removed (PR " + pr.URL + ")"})
					break
				}
			}
		case it.Type == "PullRequest":
			pr, err := prs.get(it.URL)
			if err != nil {
				return err
			}
			if pr == nil || pr.Author.Login != s.login || pr.State != "CLOSED" {
				continue
			}
			for _, ref := range pr.ClosingIssues {
				if _, onBoard := board.Find(items, ref.URL); onBoard {
					continue
				}
				issue, err := gh.IssueView("", ref.URL)
				if err != nil {
					return err
				}
				if issue.State != "OPEN" {
					continue
				}
				open := false
				for _, other := range issue.ClosedBy {
					o, err := prs.get(other.URL)
					if err != nil {
						return err
					}
					open = open || (o != nil && o.State == "OPEN")
				}
				if open {
					continue
				}
				if err := s.board.Track(issue.URL, board.Todo); err != nil {
					return err
				}
				rows = append(rows, syncRow{issue.URL, "-", string(board.Todo)})
			}
		}
	}

	if len(rows) == 0 {
		s.theme.Successf("Board is already in step")
	}
	for _, r := range rows {
		fmt.Printf("%-60s %-18s -> %s\n", r.what, r.old, r.result)
	}
	for _, b := range skipped {
		fmt.Printf("skipped %s (no PR)\n", b)
	}
	return nil
}
