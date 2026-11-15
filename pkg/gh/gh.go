// Package gh wraps the `gh` binary. Every GitHub call in gh-board goes
// through here, so --dry-run is implemented once: reads always run, writes
// are printed and skipped.
package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/SamuelBisberg/gh-board/pkg/progress"
)

// DryRun makes Write print the planned call instead of running it.
var DryRun bool

// Out receives dry-run output.
var Out io.Writer = os.Stdout

// labels name the slow calls for the progress spinner, by their first two
// arguments.
var labels = map[string]string{
	"api user":             "Checking your account",
	"api graphql":          "Reading the board",
	"repo view":            "Reading the repository",
	"pr view":              "Reading the pull request",
	"pr create":            "Creating the pull request",
	"pr ready":             "Marking the pull request ready",
	"pr edit":              "Requesting reviewers",
	"issue view":           "Reading the issue",
	"issue create":         "Creating the issue",
	"issue develop":        "Linking the branch to the issue",
	"project view":         "Opening the board",
	"project field-list":   "Reading the board's fields",
	"project item-list":    "Reading the board's cards",
	"project item-add":     "Adding to the board",
	"project item-edit":    "Updating the card",
	"project item-delete":  "Removing the card",
	"project create":       "Creating the board",
	"project field-create": "Creating the Status field",
	"project list":         "Listing projects",
}

func label(args []string) string {
	if len(args) >= 2 {
		if l, ok := labels[args[0]+" "+args[1]]; ok {
			return l
		}
	}
	return "Running gh " + strings.Join(args[:min(len(args), 2)], " ")
}

func run(args ...string) (string, error) {
	var out string
	err := progress.Run(label(args), func() error {
		cmd := exec.Command("gh", args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("gh %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
		}
		out = strings.TrimSpace(stdout.String())
		return nil
	})
	return out, err
}

// Read runs a gh call that does not change anything.
func Read(args ...string) (string, error) { return run(args...) }

// Write runs a gh call that changes GitHub. In dry-run mode it prints the
// call and returns an empty result.
func Write(args ...string) (string, error) {
	if DryRun {
		fmt.Fprintf(Out, "+ gh %s\n", strings.Join(args, " "))
		return "", nil
	}
	return run(args...)
}

// ReadJSON runs a read call and decodes its JSON output into v.
func ReadJSON(v any, args ...string) error {
	out, err := Read(args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		return fmt.Errorf("decoding output of gh %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

// Login returns the logged-in user's login.
func Login() (string, error) {
	return Read("api", "user", "--jq", ".login")
}

type Repo struct {
	NameWithOwner string `json:"nameWithOwner"`
	DefaultBranch struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
}

// CurrentRepo returns the repo of the current directory (empty ref), or the
// named owner/repo.
func CurrentRepo(ref string) (*Repo, error) {
	args := []string{"repo", "view"}
	if ref != "" {
		args = append(args, ref)
	}
	args = append(args, "--json", "nameWithOwner,defaultBranchRef")
	var r Repo
	if err := ReadJSON(&r, args...); err != nil {
		return nil, err
	}
	return &r, nil
}

type Login_ struct {
	Login string `json:"login"`
}

type IssueRef struct {
	Number     int    `json:"number"`
	URL        string `json:"url"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

type PR struct {
	Number         int        `json:"number"`
	URL            string     `json:"url"`
	Title          string     `json:"title"`
	IsDraft        bool       `json:"isDraft"`
	State          string     `json:"state"`
	ReviewDecision string     `json:"reviewDecision"`
	HeadRefName    string     `json:"headRefName"`
	BaseRefName    string     `json:"baseRefName"`
	Author         Login_     `json:"author"`
	ReviewRequests []Login_   `json:"reviewRequests"`
	ClosingIssues  []IssueRef `json:"closingIssuesReferences"`
}

const prFields = "number,url,title,isDraft,state,reviewDecision,headRefName,baseRefName,author,reviewRequests,closingIssuesReferences"

// PRView looks up a PR by branch name, number or URL. It returns nil, nil
// when the branch has no PR.
func PRView(ref string) (*PR, error) {
	var pr PR
	err := ReadJSON(&pr, "pr", "view", ref, "--json", prFields)
	if err != nil {
		if strings.Contains(err.Error(), "no pull requests found") {
			return nil, nil
		}
		return nil, err
	}
	return &pr, nil
}

type Issue struct {
	Number   int      `json:"number"`
	URL      string   `json:"url"`
	Title    string   `json:"title"`
	State    string   `json:"state"`
	Author   Login_   `json:"author"`
	ClosedBy []PRRef  `json:"closedByPullRequestsReferences"`
	Assign   []Login_ `json:"assignees"`
}

type PRRef struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// IssueView looks up an issue by number (in repo, or the current repo if
// repo is empty) or by URL.
func IssueView(repo, ref string) (*Issue, error) {
	args := []string{"issue", "view", ref}
	if repo != "" {
		args = append(args, "-R", repo)
	}
	args = append(args, "--json", "number,url,title,state,author,assignees,closedByPullRequestsReferences")
	var i Issue
	if err := ReadJSON(&i, args...); err != nil {
		return nil, err
	}
	return &i, nil
}

// lastURL extracts the PR/issue URL from `gh ... create` stdout.
func lastURL(out string) string {
	lines := strings.Split(out, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); strings.HasPrefix(l, "http") {
			return l
		}
	}
	return ""
}

// IssueCreate creates an issue assigned to the logged-in user and returns
// its URL.
func IssueCreate(repo, title, body string) (string, error) {
	out, err := Write("issue", "create", "-R", repo, "--title", title, "--body", body, "--assignee", "@me")
	return lastURL(out), err
}

// IssueDevelop creates a branch linked to the issue on the remote.
func IssueDevelop(number int, branch, base string) error {
	_, err := Write("issue", "develop", fmt.Sprint(number), "--name", branch, "--base", base)
	return err
}

// PRCreateOpts are the inputs to PRCreate.
type PRCreateOpts struct {
	Title, Body, Base, Head string
	Draft                   bool
	Reviewers               []string
}

// PRCreate opens a PR assigned to the logged-in user and returns its URL.
func PRCreate(o PRCreateOpts) (string, error) {
	args := []string{"pr", "create", "--title", o.Title, "--body", o.Body, "--base", o.Base, "--head", o.Head, "--assignee", "@me"}
	if o.Draft {
		args = append(args, "--draft")
	}
	if len(o.Reviewers) > 0 {
		args = append(args, "--reviewer", strings.Join(o.Reviewers, ","))
	}
	out, err := Write(args...)
	return lastURL(out), err
}

// PRReady marks a draft PR ready for review.
func PRReady(ref string) error {
	_, err := Write("pr", "ready", ref)
	return err
}

// PRAddReviewers requests reviews from the given logins.
func PRAddReviewers(ref string, logins []string) error {
	if len(logins) == 0 {
		return nil
	}
	_, err := Write("pr", "edit", ref, "--add-reviewer", strings.Join(logins, ","))
	return err
}
