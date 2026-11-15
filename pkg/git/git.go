// Package git wraps the system git binary with the porcelain operations
// gh-board needs. It never handles credentials itself - it relies on
// whatever helper `gh auth login` already configured.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/SamuelBisberg/gh-board/pkg/progress"
)

// run executes git with the given args in the current working directory,
// returning stdout with its trailing newline removed. On failure the error
// includes stderr.
func run(args ...string) (string, error) {
	var out string
	err := progress.Run(label(args), func() error {
		cmd := exec.Command("git", args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
		}
		out = strings.TrimRight(stdout.String(), "\n")
		return nil
	})
	return out, err
}

// label names the network-touching git calls for the progress spinner;
// local ones are fast enough that the spinner never appears for them.
func label(args []string) string {
	switch args[0] {
	case "fetch":
		return "Fetching origin"
	case "push":
		return "Pushing"
	}
	return "Running git " + args[0]
}

// IsRepo reports whether the current directory is inside a git work tree.
func IsRepo() bool {
	out, err := run("rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// IsDirty reports whether the working tree has uncommitted changes.
func IsDirty() (bool, error) {
	out, err := run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// CurrentBranch returns the name of the currently checked-out branch.
func CurrentBranch() (string, error) {
	return run("rev-parse", "--abbrev-ref", "HEAD")
}

// DryRun makes the mutating helpers print the planned call instead of
// running it.
var DryRun bool

func mutate(args ...string) (string, error) {
	if DryRun {
		fmt.Printf("+ git %s\n", strings.Join(args, " "))
		return "", nil
	}
	return run(args...)
}

// Fetch fetches origin.
func Fetch() error {
	_, err := run("fetch", "origin")
	return err
}

// Branches lists local branch names.
func Branches() ([]string, error) {
	out, err := run("for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

// CreateBranchFrom creates and checks out branch from start, keeping
// uncommitted changes.
func CreateBranchFrom(branch, start string) error {
	_, err := mutate("checkout", "-b", branch, "--no-track", start)
	return err
}

// EmptyCommit makes a commit with no changes.
func EmptyCommit(message string) error {
	_, err := mutate("commit", "--allow-empty", "-m", message)
	return err
}

// Push runs git push with args, defaulting to `-u origin HEAD`.
func Push(args ...string) error {
	if len(args) == 0 {
		args = []string{"-u", "origin", "HEAD"}
	}
	_, err := mutate(append([]string{"push"}, args...)...)
	return err
}

// CommitsAhead counts commits on HEAD that are not on ref.
func CommitsAhead(ref string) (int, error) {
	out, err := run("rev-list", "--count", ref+"..HEAD")
	if err != nil {
		return 0, err
	}
	var n int
	_, err = fmt.Sscanf(out, "%d", &n)
	return n, err
}

// RepoRoot returns the top-level directory of the work tree.
func RepoRoot() (string, error) {
	return run("rev-parse", "--show-toplevel")
}
