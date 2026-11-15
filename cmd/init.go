package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
	"github.com/SamuelBisberg/gh-board/pkg/tui"
)

func newInitCmd() *cobra.Command {
	var org string
	var project int
	c := &cobra.Command{
		Use:   "init",
		Short: "Pick the Board, check it against board.yml and write the config",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runInit(org, project) },
	}
	c.Flags().StringVar(&org, "org", "", "organisation that owns the Board")
	c.Flags().IntVar(&project, "project", 0, "project number (skips the picker)")
	return c
}

type liveBoard struct {
	Fields struct {
		Nodes []struct {
			Name    string `json:"name"`
			Options []struct {
				Name string `json:"name"`
			} `json:"options"`
		} `json:"nodes"`
	} `json:"fields"`
	Views struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"views"`
	Workflows struct {
		Nodes []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"nodes"`
	} `json:"workflows"`
}

const liveQuery = `query($o:String!,$n:Int!){organization(login:$o){projectV2(number:$n){` +
	`fields(first:50){nodes{... on ProjectV2SingleSelectField{name options{name}}}}` +
	`views(first:20){nodes{name}} workflows(first:30){nodes{name enabled}}}}}`

func readLive(org string, number int) (board.ViewState, error) {
	var r struct {
		Data struct {
			Organization struct {
				ProjectV2 liveBoard `json:"projectV2"`
			} `json:"organization"`
		} `json:"data"`
	}
	if err := gh.ReadJSON(&r, "api", "graphql", "-f", "query="+liveQuery, "-f", "o="+org, "-F", "n="+strconv.Itoa(number)); err != nil {
		return board.ViewState{}, err
	}
	p := r.Data.Organization.ProjectV2
	st := board.ViewState{Workflows: map[string]bool{}}
	for _, f := range p.Fields.Nodes {
		if f.Name == "Status" {
			st.StatusOptions = []string{}
			for _, o := range f.Options {
				st.StatusOptions = append(st.StatusOptions, o.Name)
			}
		}
	}
	for _, v := range p.Views.Nodes {
		st.Views = append(st.Views, v.Name)
	}
	for _, w := range p.Workflows.Nodes {
		st.Workflows[w.Name] = w.Enabled
	}
	return st, nil
}

func runInit(orgFlag string, projectFlag int) error {
	if err := checkAuth(); err != nil {
		return err
	}
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	def, err := board.LoadDefinition()
	if err != nil {
		return err
	}
	if orgFlag != "" {
		cfg.Board.Org = orgFlag
	}
	if projectFlag > 0 {
		cfg.Board.Project = projectFlag
	} else if tui.IsInteractive() {
		var list struct {
			Projects []struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"projects"`
		}
		if err := gh.ReadJSON(&list, "project", "list", "--owner", cfg.Board.Org, "--format", "json"); err != nil {
			return fmt.Errorf("listing projects: %w (try `gh auth refresh -s project -s read:org`)", err)
		}
		labels := []string{"Create a new board"}
		for _, p := range list.Projects {
			labels = append(labels, fmt.Sprintf("#%d %s", p.Number, p.Title))
		}
		idx, err := tui.PickOne("Which board?", labels)
		if err != nil {
			return err
		}
		if idx > 0 {
			cfg.Board.Project = list.Projects[idx-1].Number
		} else {
			cfg.Board.Project = 0
		}
	}

	if cfg.Board.Project == 0 {
		login, err := gh.Login()
		if err != nil {
			return err
		}
		title := strings.ReplaceAll(def.Board.Title, "{{login}}", login)
		var created struct {
			Number int `json:"number"`
		}
		out, err := gh.Write("project", "create", "--owner", cfg.Board.Org, "--title", title, "--format", "json")
		if err != nil {
			return fmt.Errorf("creating the board: %w", err)
		}
		if out != "" {
			if err := jsonUnmarshal(out, &created); err != nil {
				return err
			}
		}
		cfg.Board.Project = created.Number
		theme.Successf("Created board %q (#%d)", title, created.Number)
	}

	live, err := readLive(cfg.Board.Org, cfg.Board.Project)
	if err != nil {
		return fmt.Errorf("reading board %s #%d: %w", cfg.Board.Org, cfg.Board.Project, err)
	}
	if live.StatusOptions == nil {
		if _, err := gh.Write("project", "field-create", strconv.Itoa(cfg.Board.Project), "--owner", cfg.Board.Org,
			"--name", "Status", "--data-type", "SINGLE_SELECT",
			"--single-select-options", strings.Join(def.StatusOptionNames(), ",")); err != nil {
			return fmt.Errorf("creating the Status field: %w", err)
		}
		theme.Successf("Created the Status field")
		live.StatusOptions = def.StatusOptionNames()
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	theme.Successf("Using board %s #%d", cfg.Board.Org, cfg.Board.Project)

	problems := def.Check(live)
	for _, p := range problems {
		theme.Warningf("%s", p)
	}
	fmt.Println("Verify once in the board's Workflows settings:")
	for _, w := range def.Workflows {
		if w.Enabled && w.ExpectedAction != "" {
			fmt.Printf("  %s -> %s\n", w.Name, w.ExpectedAction)
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("the board does not match board.yml (%d problem(s)); fix the above and re-run `gh board init`", len(problems))
	}
	theme.Successf("The board matches board.yml")
	return nil
}
