package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SamuelBisberg/gh-board/pkg/config"
)

var fakeGHBinary string

// TestMain builds the fake `gh` (cmd/testdata/fakegh) once for all tests.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fakegh")
	if err != nil {
		panic(err)
	}
	fakeGHBinary = filepath.Join(dir, "gh")
	if out, err := exec.Command("go", "build", "-o", fakeGHBinary, "./testdata/fakegh").CombinedOutput(); err != nil {
		os.Stderr.Write(out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type rule struct {
	Match  string // substring of the space-joined args
	Stdout string
	Fail   string // non-empty: exit 1 with this on stderr
}

type fakeGH struct {
	t   *testing.T
	log string
}

// useFakeGH installs the fake gh with rules (first match wins), plus the
// baseline rules every board command needs, and isolates auth.
func useFakeGH(t *testing.T, rules ...rule) *fakeGH {
	t.Helper()
	isolateGH(t)
	t.Setenv("GH_TOKEN", "test-token")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, _ := config.Load()
	cfg.Board.Owner, cfg.Board.Project = "test-owner", 8
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "rules.json")
	data, _ := json.Marshal(append(rules, baseRules()...))
	os.WriteFile(rulesPath, data, 0o644)

	if err := os.Symlink(fakeGHBinary, filepath.Join(dir, "gh")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_FAKE_RULES", rulesPath)
	logPath := filepath.Join(dir, "calls.log")
	t.Setenv("GH_FAKE_LOG", logPath)
	return &fakeGH{t: t, log: logPath}
}

func baseRules() []rule {
	return []rule{
		{Match: "api user", Stdout: "me"},
		{Match: "repo view", Stdout: `{"nameWithOwner":"o/r","defaultBranchRef":{"name":"main"}}`},
		{Match: "project view", Stdout: `{"id":"PID"}`},
		{Match: "project field-list", Stdout: `{"fields":[{"id":"FID","name":"Status","options":[
			{"id":"opt-todo","name":"Todo"},{"id":"opt-progress","name":"In Progress"},
			{"id":"opt-review","name":"Needs Review"},{"id":"opt-changes","name":"Changes Requested"},
			{"id":"opt-approved","name":"Approved & Merged"},{"id":"opt-closed","name":"Closed"}]}]}`},
		{Match: "project item-list", Stdout: `{"items":[]}`},
		{Match: "project item-add", Stdout: `{"id":"NEWITEM"}`},
	}
}

// calls returns every gh invocation so far, args joined by spaces.
func (f *fakeGH) calls() []string {
	data, _ := os.ReadFile(f.log)
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var args []string
		if json.Unmarshal([]byte(l), &args) == nil {
			out = append(out, strings.Join(args, " "))
		}
	}
	return out
}

// has reports whether some call contains every given substring.
func (f *fakeGH) has(subs ...string) bool {
	for _, c := range f.calls() {
		ok := true
		for _, s := range subs {
			ok = ok && strings.Contains(c, s)
		}
		if ok {
			return true
		}
	}
	return false
}

func (f *fakeGH) mustHave(subs ...string) {
	f.t.Helper()
	if !f.has(subs...) {
		f.t.Errorf("no gh call containing %q\ncalls:\n%s", subs, strings.Join(f.calls(), "\n"))
	}
}

func (f *fakeGH) mustNotHave(subs ...string) {
	f.t.Helper()
	if f.has(subs...) {
		f.t.Errorf("unexpected gh call containing %q\ncalls:\n%s", subs, strings.Join(f.calls(), "\n"))
	}
}

func isolateGH(t *testing.T) {
	t.Helper()
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("GH_PATH", "/nonexistent/gh")
	for _, v := range []string{"GH_HOST", "GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"} {
		t.Setenv(v, "")
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// setupRepo creates a repo with a bare "origin", one commit pushed on main,
// and chdirs the test into it.
func setupRepo(t *testing.T) (repo, remote string) {
	t.Helper()
	remote = t.TempDir()
	runGit(t, remote, "init", "--bare", "-b", "main")
	repo = t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test")
	runGit(t, repo, "config", "commit.gpgsign", "false")
	runGit(t, repo, "remote", "add", "origin", remote)
	os.WriteFile(filepath.Join(repo, "file.txt"), []byte("hi"), 0o644)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "init")
	runGit(t, repo, "push", "-u", "origin", "main")
	t.Chdir(repo)
	return repo, remote
}

// featureBranch checks out a branch with one commit ahead of main.
func featureBranch(t *testing.T, repo, name string) {
	t.Helper()
	runGit(t, repo, "checkout", "-b", name)
	runGit(t, repo, "commit", "--allow-empty", "-m", "work")
}
