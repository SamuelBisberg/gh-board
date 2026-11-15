package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SamuelBisberg/gh-board/pkg/config"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/git"
)

func prJSON(url string, draft bool, state, decision, author string) string {
	return fmt.Sprintf(`{"number":1,"url":%q,"title":"t","isDraft":%t,"state":%q,"reviewDecision":%q,
		"headRefName":"feat","baseRefName":"main","author":{"login":%q},"reviewRequests":[],"closingIssuesReferences":[]}`,
		url, draft, state, decision, author)
}

func TestNewCmdRegistersEverySubcommand(t *testing.T) {
	root := NewRootCmd()
	for _, name := range []string{"todo", "new", "push", "open", "review", "team", "sync", "init", "config"} {
		found := false
		for _, c := range root.Commands() {
			found = found || c.Name() == name
		}
		if !found {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}

func TestTodoCreatesIssueCard(t *testing.T) {
	f := useFakeGH(t, rule{Match: "issue create", Stdout: "https://github.com/o/r/issues/7\n"})
	setupRepo(t)

	if err := runTodo("Upgrade filtering", "", ""); err != nil {
		t.Fatal(err)
	}
	f.mustHave("issue create", "-R o/r", "--title Upgrade filtering", "--assignee @me", "## Summary")
	f.mustHave("project item-add", "--url https://github.com/o/r/issues/7")
	f.mustHave("item-edit", "--id NEWITEM", "opt-todo")
}

func TestTodoUsesRepoTemplateThenExplicitBody(t *testing.T) {
	f := useFakeGH(t, rule{Match: "issue create", Stdout: "https://github.com/o/r/issues/7\n"})
	repo, _ := setupRepo(t)
	os.MkdirAll(filepath.Join(repo, ".github"), 0o755)
	os.WriteFile(filepath.Join(repo, ".github", "ISSUE_TEMPLATE.md"), []byte("REPO TEMPLATE for {{title}}"), 0o644)

	if err := runTodo("Thing", "", ""); err != nil {
		t.Fatal(err)
	}
	f.mustHave("issue create", "REPO TEMPLATE for Thing")

	if err := runTodo("Other", "explicit", ""); err != nil {
		t.Fatal(err)
	}
	f.mustHave("issue create", "--title Other", "--body explicit")
}

func TestTodoNeedsARepo(t *testing.T) {
	useFakeGH(t)
	t.Chdir(t.TempDir())
	if err := runTodo("x", "", ""); err == nil {
		t.Fatal("want an error with no repo, no todo.repo and no -R")
	}
}

func TestNewFromIssueReplacesIssueCard(t *testing.T) {
	f := useFakeGH(t,
		rule{Match: "issue view", Stdout: `{"number":12,"url":"https://github.com/o/r/issues/12","title":"Upgrade filtering","state":"OPEN","author":{"login":"me"}}`},
		rule{Match: "pr create", Stdout: "https://github.com/o/r/pull/30\n"},
		rule{Match: "project item-list", Stdout: `{"items":[{"id":"ISSUECARD","status":"Todo","content":{"type":"Issue","number":12,"repository":"o/r","url":"https://github.com/o/r/issues/12","title":"Upgrade filtering"}}]}`},
	)
	repo, _ := setupRepo(t)
	// `gh issue develop` creates the linked branch on the remote; fake it.
	runGit(t, repo, "push", "origin", "main:task/upgrade-filtering")

	if err := runNew("", "", "", 12); err != nil {
		t.Fatal(err)
	}
	f.mustHave("issue develop 12", "--name task/upgrade-filtering")
	f.mustHave("pr create", "--draft", "--title task/upgrade filtering", "--head task/upgrade-filtering", "Closes #12")
	f.mustHave("item-edit", "--id NEWITEM", "opt-progress")
	f.mustHave("project item-delete", "--id ISSUECARD")
	if got := runGit(t, repo, "rev-parse", "--abbrev-ref", "HEAD"); got != "task/upgrade-filtering" {
		t.Errorf("branch = %q", got)
	}
	if got := runGit(t, repo, "log", "-1", "--format=%s"); got != "chore: start task/upgrade-filtering" {
		t.Errorf("last commit = %q", got)
	}
}

func TestOpenReadiesDraftAndRequestsReviewers(t *testing.T) {
	f := useFakeGH(t, rule{Match: "pr view", Stdout: prJSON("https://github.com/o/r/pull/1", true, "OPEN", "", "me")})
	repo, _ := setupRepo(t)
	featureBranch(t, repo, "feat")
	cfg, _ := config.Load()
	cfg.Team = []config.Teammate{{Alias: "ana", Login: "ana-lopez-1234"}, {Alias: "me", Login: "me"}}
	cfg.Save()

	if err := runOpen("", []string{"ana", "me"}, false); err != nil {
		t.Fatal(err)
	}
	f.mustHave("pr ready")
	f.mustHave("pr edit", "--add-reviewer ana-lopez-1234")
	f.mustNotHave("--add-reviewer ana-lopez-1234,me")
	f.mustHave("item-edit", "opt-review")
}

func TestOpenUsesDefaultReviewersAndNoReviewersFlag(t *testing.T) {
	f := useFakeGH(t, rule{Match: "pr view", Stdout: prJSON("https://github.com/o/r/pull/1", true, "OPEN", "", "me")})
	repo, _ := setupRepo(t)
	featureBranch(t, repo, "feat")
	cfg, _ := config.Load()
	cfg.Team = []config.Teammate{{Alias: "ana", Login: "ana-lopez-1234"}}
	cfg.Reviewers.Default = []string{"ana"}
	cfg.Save()

	if err := runOpen("", nil, false); err != nil {
		t.Fatal(err)
	}
	f.mustHave("pr edit", "--add-reviewer ana-lopez-1234")
}

func TestOpenNoReviewers(t *testing.T) {
	f := useFakeGH(t, rule{Match: "pr view", Stdout: prJSON("https://github.com/o/r/pull/1", true, "OPEN", "", "me")})
	repo, _ := setupRepo(t)
	featureBranch(t, repo, "feat")
	cfg, _ := config.Load()
	cfg.Team = []config.Teammate{{Alias: "ana", Login: "ana"}}
	cfg.Reviewers.Default = []string{"ana"}
	cfg.Save()

	if err := runOpen("", nil, true); err != nil {
		t.Fatal(err)
	}
	f.mustHave("pr ready")
	f.mustNotHave("--add-reviewer")
}

func TestOpenAlreadyReadyDoesNotRenotify(t *testing.T) {
	f := useFakeGH(t, rule{Match: "pr view", Stdout: prJSON("https://github.com/o/r/pull/1", false, "OPEN", "CHANGES_REQUESTED", "me")})
	repo, _ := setupRepo(t)
	featureBranch(t, repo, "feat")
	cfg, _ := config.Load()
	cfg.Team = []config.Teammate{{Alias: "ana", Login: "ana"}}
	cfg.Reviewers.Default = []string{"ana"}
	cfg.Save()

	if err := runOpen("", nil, false); err != nil {
		t.Fatal(err)
	}
	f.mustNotHave("pr ready")
	f.mustNotHave("--add-reviewer")
	f.mustHave("item-edit", "opt-changes")
}

func TestOpenCreatesReadyPRWhenNone(t *testing.T) {
	f := useFakeGH(t,
		rule{Match: "pr view", Fail: "no pull requests found for branch"},
		rule{Match: "pr create", Stdout: "https://github.com/o/r/pull/9\n"},
	)
	repo, _ := setupRepo(t)
	featureBranch(t, repo, "refactor-wf-inputs")

	if err := runOpen("", nil, true); err != nil {
		t.Fatal(err)
	}
	f.mustHave("pr create", "--title refactor wf inputs", "## Summary")
	f.mustNotHave("--draft")
	f.mustHave("item-edit", "opt-review")
}

func TestOpenRefusals(t *testing.T) {
	t.Run("default branch", func(t *testing.T) {
		useFakeGH(t)
		setupRepo(t)
		if err := runOpen("", nil, true); err == nil {
			t.Fatal("want an error on the default branch")
		}
	})
	t.Run("merged PR", func(t *testing.T) {
		useFakeGH(t, rule{Match: "pr view", Stdout: prJSON("https://github.com/o/r/pull/1", false, "MERGED", "", "me")})
		repo, _ := setupRepo(t)
		featureBranch(t, repo, "feat")
		if err := runOpen("", nil, true); err == nil {
			t.Fatal("want an error for a merged PR")
		}
	})
	t.Run("no commits ahead", func(t *testing.T) {
		useFakeGH(t, rule{Match: "pr view", Fail: "no pull requests found"})
		repo, _ := setupRepo(t)
		runGit(t, repo, "checkout", "-b", "feat")
		if err := runOpen("", nil, true); err == nil {
			t.Fatal("want an error with no commits ahead")
		}
	})
}

func TestPushOpensDraftPRWhenNone(t *testing.T) {
	f := useFakeGH(t,
		rule{Match: "pr view", Fail: "no pull requests found"},
		rule{Match: "pr create", Stdout: "https://github.com/o/r/pull/9\n"},
	)
	repo, remote := setupRepo(t)
	featureBranch(t, repo, "feat")

	if err := runPush(nil); err != nil {
		t.Fatal(err)
	}
	f.mustHave("pr create", "--draft")
	f.mustHave("item-edit", "opt-progress")
	if !strings.Contains(runGit(t, remote, "branch", "--list"), "feat") {
		t.Error("branch was not pushed")
	}
}

func TestPushRefusesDefaultBranch(t *testing.T) {
	useFakeGH(t)
	setupRepo(t)
	if err := runPush(nil); err == nil {
		t.Fatal("want an error on the default branch")
	}
}

func TestReviewSkipsPendingAndRefusesDrafts(t *testing.T) {
	pr := strings.Replace(prJSON("https://github.com/o/r/pull/1", false, "OPEN", "", "me"),
		`"reviewRequests":[]`, `"reviewRequests":[{"login":"ana"}]`, 1)
	f := useFakeGH(t, rule{Match: "pr view", Stdout: pr})
	repo, _ := setupRepo(t)
	featureBranch(t, repo, "feat")

	if err := runReview([]string{"ana", "bob"}); err != nil {
		t.Fatal(err)
	}
	f.mustHave("pr edit", "--add-reviewer bob")
	f.mustNotHave("--add-reviewer ana")
	f.mustNotHave("item-edit")

	g := useFakeGH(t, rule{Match: "pr view", Stdout: prJSON("https://github.com/o/r/pull/1", true, "OPEN", "", "me")})
	if err := runReview([]string{"bob"}); err == nil {
		t.Fatal("want an error for a draft PR")
	}
	g.mustNotHave("--add-reviewer")
}

func card(id, typ string, n int, status string) string {
	return fmt.Sprintf(`{"id":%q,"status":%q,"content":{"type":%q,"number":%d,"repository":"o/r","url":"https://github.com/o/r/%s/%d","title":"t"}}`,
		id, status, typ, n, map[string]string{"PullRequest": "pull", "Issue": "issues"}[typ], n)
}

func TestSyncDerivesStatusFromPR(t *testing.T) {
	const base = "https://github.com/o/r/pull/"
	items := strings.Join([]string{
		card("c1", "PullRequest", 1, "Todo"),         // draft -> In Progress
		card("c2", "PullRequest", 2, "In Progress"),  // ready -> Needs Review
		card("c3", "PullRequest", 3, "Needs Review"), // changes requested
		card("c4", "PullRequest", 4, "Needs Review"), // approved while open
		card("c5", "PullRequest", 5, "Needs Review"), // merged
		card("c6", "PullRequest", 6, "Needs Review"), // closed unmerged: ignored
		card("c7", "PullRequest", 7, "Needs Review"), // someone else's: untouched
		card("c8", "PullRequest", 8, "Closed"),       // Closed: untouched
		card("c9", "PullRequest", 9, "Needs Review"), // already correct
	}, ",")
	f := useFakeGH(t,
		rule{Match: "pr view " + base + "1 ", Stdout: prJSON(base+"1", true, "OPEN", "", "me")},
		rule{Match: "pr view " + base + "2 ", Stdout: prJSON(base+"2", false, "OPEN", "REVIEW_REQUIRED", "me")},
		rule{Match: "pr view " + base + "3 ", Stdout: prJSON(base+"3", false, "OPEN", "CHANGES_REQUESTED", "me")},
		rule{Match: "pr view " + base + "4 ", Stdout: prJSON(base+"4", false, "OPEN", "APPROVED", "me")},
		rule{Match: "pr view " + base + "5 ", Stdout: prJSON(base+"5", false, "MERGED", "APPROVED", "me")},
		rule{Match: "pr view " + base + "6 ", Stdout: prJSON(base+"6", false, "CLOSED", "", "me")},
		rule{Match: "pr view " + base + "7 ", Stdout: prJSON(base+"7", false, "OPEN", "APPROVED", "someone")},
		rule{Match: "pr view " + base + "9 ", Stdout: prJSON(base+"9", false, "OPEN", "", "me")},
		rule{Match: "pr view", Fail: "no pull requests found"},
		rule{Match: "project item-list", Stdout: `{"items":[` + items + `]}`},
	)
	setupRepo(t)

	if err := runSync(false); err != nil {
		t.Fatal(err)
	}
	for id, opt := range map[string]string{"c1": "opt-progress", "c2": "opt-review", "c3": "opt-changes", "c4": "opt-approved", "c5": "opt-approved"} {
		f.mustHave("item-edit", "--id "+id+" ", opt)
	}
	for _, id := range []string{"c6", "c7", "c8", "c9"} {
		f.mustNotHave("item-edit", "--id "+id+" ")
	}
	f.mustNotHave("opt-closed")
}

func TestSyncTracksLocalBranchPRsOfMine(t *testing.T) {
	const url = "https://github.com/o/r/pull/21"
	f := useFakeGH(t,
		rule{Match: "pr view feat ", Stdout: prJSON(url, false, "OPEN", "", "me")},
		rule{Match: "pr view", Fail: "no pull requests found"},
	)
	repo, _ := setupRepo(t)
	featureBranch(t, repo, "feat")

	if err := runSync(true); err != nil {
		t.Fatal(err)
	}
	f.mustHave("project item-add", "--url "+url)
	f.mustHave("item-edit", "opt-review")

	// Idempotent: once the card is correct, a second sync edits nothing.
	g := useFakeGH(t,
		rule{Match: "pr view", Stdout: prJSON(url, false, "OPEN", "", "me")},
		rule{Match: "project item-list", Stdout: `{"items":[` + card("c1", "PullRequest", 21, "Needs Review") + `]}`},
	)
	if err := runSync(true); err != nil {
		t.Fatal(err)
	}
	g.mustNotHave("item-edit")
	g.mustNotHave("item-add")
}

func TestSyncDoesNotTrackOthersOrMergedBranches(t *testing.T) {
	f := useFakeGH(t, rule{Match: "pr view", Stdout: prJSON("https://github.com/o/r/pull/3", false, "MERGED", "", "me")})
	repo, _ := setupRepo(t)
	featureBranch(t, repo, "feat")
	if err := runSync(true); err != nil {
		t.Fatal(err)
	}
	f.mustNotHave("item-add")
}

func TestSyncReconcilesIssueCards(t *testing.T) {
	const issueURL = "https://github.com/o/r/issues/12"
	const prURL = "https://github.com/o/r/pull/30"
	issue := `{"number":12,"url":"` + issueURL + `","title":"t","state":"OPEN","author":{"login":"me"},"closedByPullRequestsReferences":[{"number":30,"url":"` + prURL + `"}]}`

	t.Run("removes the issue card once my PR closes it", func(t *testing.T) {
		f := useFakeGH(t,
			rule{Match: "issue view", Stdout: issue},
			rule{Match: "pr view", Stdout: prJSON(prURL, true, "OPEN", "", "me")},
			rule{Match: "project item-list", Stdout: `{"items":[` + card("ic", "Issue", 12, "Todo") + `]}`},
		)
		setupRepo(t)
		if err := runSync(true); err != nil {
			t.Fatal(err)
		}
		f.mustHave("project item-delete", "--id ic")
	})

	t.Run("restores the issue to Todo when its PR closed unmerged", func(t *testing.T) {
		closedPR := strings.Replace(prJSON(prURL, false, "CLOSED", "", "me"), `"closingIssuesReferences":[]`,
			`"closingIssuesReferences":[{"number":12,"url":"`+issueURL+`"}]`, 1)
		f := useFakeGH(t,
			rule{Match: "issue view", Stdout: issue},
			rule{Match: "pr view", Stdout: closedPR},
			rule{Match: "project item-list", Stdout: `{"items":[` + card("pc", "PullRequest", 30, "Closed") + `]}`},
		)
		setupRepo(t)
		if err := runSync(true); err != nil {
			t.Fatal(err)
		}
		f.mustHave("project item-add", "--url "+issueURL)
		f.mustHave("item-edit", "opt-todo")
		f.mustNotHave("item-edit", "--id pc")
	})

	t.Run("keeps it off the Board when another PR is open", func(t *testing.T) {
		const otherURL = "https://github.com/o/r/pull/31"
		two := strings.Replace(issue, `[{"number":30,`, `[{"number":31,"url":"`+otherURL+`"},{"number":30,`, 1)
		closedPR := strings.Replace(prJSON(prURL, false, "CLOSED", "", "me"), `"closingIssuesReferences":[]`,
			`"closingIssuesReferences":[{"number":12,"url":"`+issueURL+`"}]`, 1)
		f := useFakeGH(t,
			rule{Match: "issue view", Stdout: two},
			rule{Match: "pr view " + otherURL, Stdout: prJSON(otherURL, false, "OPEN", "", "someone")},
			rule{Match: "pr view", Stdout: closedPR},
			rule{Match: "project item-list", Stdout: `{"items":[` + card("pc", "PullRequest", 30, "Closed") + `]}`},
		)
		setupRepo(t)
		if err := runSync(true); err != nil {
			t.Fatal(err)
		}
		f.mustNotHave("project item-add")
	})
}

func TestDryRunChangesNothing(t *testing.T) {
	f := useFakeGH(t)
	setupRepo(t)
	gh.DryRun, git.DryRun = true, true
	t.Cleanup(func() { gh.DryRun, git.DryRun = false, false })

	if err := runTodo("Dry", "", ""); err != nil {
		t.Fatal(err)
	}
	f.mustNotHave("issue create")
	f.mustNotHave("item-add")
	f.mustNotHave("item-edit")
}

func TestTeamAddValidatesOrgMembership(t *testing.T) {
	f := useFakeGH(t,
		rule{Match: "users/ana-lopez-1234", Stdout: "ana-lopez-1234"},
		rule{Match: "users/outsider", Stdout: "outsider"},
		rule{Match: "users/test-owner", Stdout: "Organization"},
		rule{Match: "orgs/test-owner/members/outsider", Fail: "404"},
	)

	if err := runTeamAdd("outsider", ""); err == nil {
		t.Fatal("want an error for a non-member")
	}
	if err := runTeamAdd("ana-lopez-1234", "ana"); err != nil {
		t.Fatal(err)
	}
	if err := runTeamAdd("ana-lopez-1234", "ANA"); err == nil {
		t.Fatal("want an error for a duplicate alias (case-insensitive)")
	}
	f.mustHave("orgs/test-owner/members/ana-lopez-1234")

	cfg, _ := config.Load()
	if len(cfg.Team) != 1 || cfg.Team[0].Alias != "ana" {
		t.Fatalf("team = %+v", cfg.Team)
	}
	if err := runTeamDefault([]string{"ana"}); err != nil {
		t.Fatal(err)
	}
	if err := runTeamRemove("ana-lopez-1234"); err != nil {
		t.Fatal(err)
	}
	cfg, _ = config.Load()
	if len(cfg.Team) != 0 || len(cfg.Reviewers.Default) != 0 {
		t.Fatalf("remove should drop the teammate and its default: %+v", cfg)
	}
}

func TestInitReportsMismatchesAndWritesConfig(t *testing.T) {
	useFakeGH(t, rule{Match: "api graphql", Stdout: `{"data":{"repositoryOwner":{"projectV2":{
		"fields":{"nodes":[{"name":"Status","options":[{"name":"Todo"},{"name":"Done"}]}]},
		"views":{"nodes":[]},
		"workflows":{"nodes":[{"name":"Item added to project","enabled":true}]}}}}}`})

	err := runInit("my-org", 3)
	if err == nil {
		t.Fatal("want a non-zero exit on mismatch")
	}
	cfg, _ := config.Load()
	if cfg.Board.Owner != "my-org" || cfg.Board.Project != 3 {
		t.Errorf("config not written: %+v", cfg.Board)
	}
}

func TestInitNeverEditsAnExistingStatusField(t *testing.T) {
	f := useFakeGH(t, rule{Match: "api graphql", Stdout: `{"data":{"repositoryOwner":{"projectV2":{
		"fields":{"nodes":[{"name":"Status","options":[{"name":"Todo"},{"name":"Done"}]}]},
		"views":{"nodes":[{"name":"PRs"}]},"workflows":{"nodes":[{"name":"Item added to project","enabled":false},{"name":"Item closed","enabled":true},{"name":"Pull request merged","enabled":true},{"name":"Code changes requested","enabled":true},{"name":"Code review approved","enabled":true},{"name":"Pull request linked to issue","enabled":true},{"name":"Auto-archive items","enabled":true},{"name":"Auto-close issue","enabled":false},{"name":"When an item in the project has sub-issues","enabled":true}]}}}}}`})
	runInit("", 8)
	f.mustNotHave("field-create")
	f.mustNotHave("field-delete")
}

func TestInitCreatesMissingStatusField(t *testing.T) {
	f := useFakeGH(t, rule{Match: "api graphql", Stdout: `{"data":{"repositoryOwner":{"projectV2":{
		"fields":{"nodes":[{}]},"views":{"nodes":[{"name":"PRs"}]},"workflows":{"nodes":[{"name":"Item added to project","enabled":false},{"name":"Item closed","enabled":true},{"name":"Pull request merged","enabled":true},{"name":"Code changes requested","enabled":true},{"name":"Code review approved","enabled":true},{"name":"Pull request linked to issue","enabled":true},{"name":"Auto-archive items","enabled":true},{"name":"Auto-close issue","enabled":false},{"name":"When an item in the project has sub-issues","enabled":true}]}}}}}`})
	if err := runInit("", 8); err != nil {
		t.Fatal(err)
	}
	f.mustHave("project field-create 8", "--name Status", "Todo,In Progress,Needs Review,Changes Requested,Approved & Merged,Closed")
}

func TestStatusIsNeverWrittenAsClosed(t *testing.T) {
	f := useFakeGH(t, rule{Match: "pr view", Stdout: prJSON("https://github.com/o/r/pull/1", false, "CLOSED", "", "me")})
	repo, _ := setupRepo(t)
	featureBranch(t, repo, "feat")
	if err := runPush(nil); err != nil {
		t.Fatal(err)
	}
	f.mustNotHave("opt-closed")
	f.mustNotHave("item-edit")
}

func TestCheckAuth(t *testing.T) {
	t.Run("not logged in", func(t *testing.T) {
		isolateGH(t)
		err := checkAuth()
		if err == nil || !strings.Contains(err.Error(), "not logged in") {
			t.Fatalf("checkAuth() = %v, want a not-logged-in error", err)
		}
	})
	t.Run("logged in via GH_TOKEN", func(t *testing.T) {
		isolateGH(t)
		t.Setenv("GH_TOKEN", "x")
		if err := checkAuth(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestConfigGetSetList(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := runConfigSet("branch.prefix", "feat"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load()
	if cfg.Branch.Prefix != "feat" {
		t.Errorf("prefix = %q", cfg.Branch.Prefix)
	}
	if err := runConfigSet("nope", "x"); err == nil {
		t.Error("want an error for an unknown key")
	}
	if err := runConfigGet("branch.prefix"); err != nil {
		t.Error(err)
	}
	if err := runConfigList(); err != nil {
		t.Error(err)
	}
}

func TestConfigTemplatesInit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	if err := runConfigTemplates(true); err != nil {
		t.Fatal(err)
	}
	dir, _ := config.TemplatesDir()
	for _, n := range []string{"issue.md", "pr.md"} {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s not written: %v", n, err)
		}
	}
}

func TestInitDefaultsToTheActiveAccount(t *testing.T) {
	f := useFakeGH(t, rule{Match: "api graphql", Stdout: `{"data":{"repositoryOwner":{"projectV2":{
		"fields":{"nodes":[{"name":"Status","options":[{"name":"Todo"},{"name":"In Progress"},{"name":"Needs Review"},{"name":"Changes Requested"},{"name":"Approved & Merged"},{"name":"Closed"}]}]},
		"views":{"nodes":[{"name":"PRs"}]},"workflows":{"nodes":[{"name":"Item added to project","enabled":false},{"name":"Item closed","enabled":true},{"name":"Pull request merged","enabled":true},{"name":"Code changes requested","enabled":true},{"name":"Code review approved","enabled":true},{"name":"Pull request linked to issue","enabled":true},{"name":"Auto-archive items","enabled":true},{"name":"Auto-close issue","enabled":false},{"name":"When an item in the project has sub-issues","enabled":true}]}}}}}`})
	cfg, _ := config.Load()
	cfg.Board.Owner, cfg.Board.Project = "", 0
	cfg.Save()

	if err := runInit("", 5); err != nil {
		t.Fatal(err)
	}
	cfg, _ = config.Load()
	if cfg.Board.Owner != "me" || cfg.Board.Project != 5 {
		t.Errorf("board = %+v, want owner of the active account (me) and project 5", cfg.Board)
	}
	f.mustHave("api graphql", "o=me")
}

func TestInitCreatesABoardForTheActiveAccount(t *testing.T) {
	f := useFakeGH(t,
		rule{Match: "project create", Stdout: `{"number":4}`},
		rule{Match: "api graphql", Stdout: `{"data":{"repositoryOwner":{"projectV2":{
			"fields":{"nodes":[{}]},"views":{"nodes":[{"name":"PRs"}]},"workflows":{"nodes":[{"name":"Item added to project","enabled":false},{"name":"Item closed","enabled":true},{"name":"Pull request merged","enabled":true},{"name":"Code changes requested","enabled":true},{"name":"Code review approved","enabled":true},{"name":"Pull request linked to issue","enabled":true},{"name":"Auto-archive items","enabled":true},{"name":"Auto-close issue","enabled":false},{"name":"When an item in the project has sub-issues","enabled":true}]}}}}}`})
	cfg, _ := config.Load()
	cfg.Board.Owner, cfg.Board.Project = "", 0
	cfg.Save()

	if err := runInit("", 0); err != nil {
		t.Fatal(err)
	}
	f.mustHave("project create", "--owner me", "--title @me's Work")
	cfg, _ = config.Load()
	if cfg.Board.Project != 4 {
		t.Errorf("project = %d, want 4", cfg.Board.Project)
	}
}

func TestChangingOwnerForgetsTheOldProjectNumber(t *testing.T) {
	useFakeGH(t, rule{Match: "project create", Stdout: `{"number":2}`},
		rule{Match: "api graphql", Stdout: `{"data":{"repositoryOwner":{"projectV2":{
			"fields":{"nodes":[{}]},"views":{"nodes":[{"name":"PRs"}]},"workflows":{"nodes":[{"name":"Item added to project","enabled":false},{"name":"Item closed","enabled":true},{"name":"Pull request merged","enabled":true},{"name":"Code changes requested","enabled":true},{"name":"Code review approved","enabled":true},{"name":"Pull request linked to issue","enabled":true},{"name":"Auto-archive items","enabled":true},{"name":"Auto-close issue","enabled":false},{"name":"When an item in the project has sub-issues","enabled":true}]}}}}}`})
	if err := runInit("someone-else", 0); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load()
	if cfg.Board.Owner != "someone-else" || cfg.Board.Project != 2 {
		t.Errorf("board = %+v", cfg.Board)
	}
}

func TestCommandsNeedAConfiguredBoard(t *testing.T) {
	useFakeGH(t)
	cfg, _ := config.Load()
	cfg.Board.Owner, cfg.Board.Project = "", 0
	cfg.Save()
	setupRepo(t)
	if err := runTodo("x", "", ""); err == nil || !strings.Contains(err.Error(), "gh board init") {
		t.Fatalf("err = %v, want a pointer to `gh board init`", err)
	}
}

func TestTeamAddOnAPersonalBoardSkipsMembership(t *testing.T) {
	f := useFakeGH(t,
		rule{Match: "users/test-owner", Stdout: "User"},
		rule{Match: "users/bob", Stdout: "bob"},
		rule{Match: "orgs/", Fail: "should not be called"})
	if err := runTeamAdd("bob", ""); err != nil {
		t.Fatal(err)
	}
	f.mustNotHave("orgs/")
}

func workflowsBoard(nodes string) rule {
	return rule{Match: "api graphql", Stdout: `{"data":{"repositoryOwner":{"projectV2":{
		"fields":{"nodes":[{}]},"views":{"nodes":[{"name":"PRs"}]},"workflows":{"nodes":[` + nodes + `]}}}}}`}
}

func TestCommandsFailWhenAStatusWorkflowIsDisabled(t *testing.T) {
	useFakeGH(t,
		rule{Match: "users/test-owner", Stdout: "Organization"},
		workflowsBoard(`{"name":"Item added to project","enabled":false},{"name":"Item closed","enabled":true},
			{"name":"Pull request merged","enabled":true},{"name":"Code changes requested","enabled":false},
			{"name":"Code review approved","enabled":true}`))
	setupRepo(t)

	err := runTodo("x", "", "")
	if err == nil {
		t.Fatal("want an error when a status workflow is disabled")
	}
	for _, want := range []string{"Code changes requested", "must be ON", "https://github.com/orgs/test-owner/projects/8/workflows", "Set Status = Changes Requested", "gh board init"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q:\n%v", want, err)
		}
	}
}

func TestCommandsFailWhenItemAddedWorkflowIsOn(t *testing.T) {
	useFakeGH(t, workflowsBoard(`{"name":"Item added to project","enabled":true},{"name":"Item closed","enabled":true},
		{"name":"Pull request merged","enabled":true},{"name":"Code changes requested","enabled":true},
		{"name":"Code review approved","enabled":true}`))
	setupRepo(t)
	err := runTodo("x", "", "")
	if err == nil || !strings.Contains(err.Error(), "must be OFF") || !strings.Contains(err.Error(), "users/test-owner/projects/8/workflows") {
		t.Fatalf("err = %v, want a must-be-OFF fix with the personal-board URL", err)
	}
}

func TestCommandsFailWhenAWorkflowIsMissing(t *testing.T) {
	useFakeGH(t, workflowsBoard(`{"name":"Item closed","enabled":true}`))
	setupRepo(t)
	err := runTodo("x", "", "")
	if err == nil || !strings.Contains(err.Error(), "was not found on the board") {
		t.Fatalf("err = %v, want a not-found message", err)
	}
}

func TestInitFailsWithFixStepsForDisabledWorkflows(t *testing.T) {
	useFakeGH(t, rule{Match: "api graphql", Stdout: `{"data":{"repositoryOwner":{"projectV2":{
		"fields":{"nodes":[{"name":"Status","options":[{"name":"Todo"},{"name":"In Progress"},{"name":"Needs Review"},{"name":"Changes Requested"},{"name":"Approved & Merged"},{"name":"Closed"}]}]},
		"views":{"nodes":[{"name":"PRs"}]},"workflows":{"nodes":[{"name":"Item closed","enabled":false}]}}}}}`})
	err := runInit("", 8)
	if err == nil || !strings.Contains(err.Error(), "no API to change workflows") {
		t.Fatalf("err = %v, want a non-zero exit explaining workflows are manual", err)
	}
}
