package config

import "testing"

func TestDefaultsAndRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Board.Org != "AWS-PLAT-LZ-SAFE" || cfg.Board.Project != 8 || cfg.Branch.Prefix != "task" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if err := cfg.Set("board.project", "12"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Set("reviewers.default", "ana, bob"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	got, _ := Load()
	if got.Board.Project != 12 || len(got.Reviewers.Default) != 2 {
		t.Fatalf("round trip lost data: %+v", got)
	}
}

func TestSetValidation(t *testing.T) {
	cfg := defaults()
	for key, bad := range map[string]string{"board.project": "x", "ui.color": "red", "todo.repo": "nope", "team": "a"} {
		if err := cfg.Set(key, bad); err == nil {
			t.Errorf("Set(%q, %q) = nil, want error", key, bad)
		}
	}
	if _, err := cfg.Get("nope"); err == nil {
		t.Error("Get(unknown) = nil error")
	}
}

func TestResolveLogin(t *testing.T) {
	cfg := defaults()
	cfg.Team = []Teammate{{Alias: "Ana", Login: "ana-lopez-1234"}}
	if got := cfg.ResolveLogin("ana"); got != "ana-lopez-1234" {
		t.Errorf("ResolveLogin(ana) = %q", got)
	}
	if got := cfg.ResolveLogin("zed"); got != "zed" {
		t.Errorf("ResolveLogin(zed) = %q", got)
	}
}
