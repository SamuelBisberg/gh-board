package git

import (
	"testing"
)

func TestIsRepo(t *testing.T) {
	newTestRepo(t)
	if !IsRepo() {
		t.Error("IsRepo() = false, want true inside a git repo")
	}

	t.Chdir(t.TempDir())
	if IsRepo() {
		t.Error("IsRepo() = true, want false outside a git repo")
	}
}

func TestCurrentBranch(t *testing.T) {
	newTestRepo(t)

	got, err := CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if got != "main" {
		t.Errorf("CurrentBranch() = %q, want %q", got, "main")
	}
}

func TestIsDirty(t *testing.T) {
	newTestRepo(t)

	dirty, err := IsDirty()
	if err != nil {
		t.Fatalf("IsDirty() error = %v", err)
	}
	if dirty {
		t.Error("IsDirty() = true, want false for a freshly committed tree")
	}

	writeFile(t, "file.txt", "changed")
	dirty, err = IsDirty()
	if err != nil {
		t.Fatalf("IsDirty() error = %v", err)
	}
	if !dirty {
		t.Error("IsDirty() = false, want true after editing a tracked file")
	}
}
