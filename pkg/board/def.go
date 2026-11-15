package board

import (
	_ "embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed board.yml
var definitionYAML []byte

// Definition is the parsed board.yml: the shape of the Board the extension
// expects.
type Definition struct {
	Version int `yaml:"version"`
	Board   struct {
		Title      string `yaml:"title"`
		Visibility string `yaml:"visibility"`
	} `yaml:"board"`
	Fields struct {
		Status struct {
			Name    string `yaml:"name"`
			Options []struct {
				Name  string `yaml:"name"`
				Color string `yaml:"color"`
			} `yaml:"options"`
		} `yaml:"status"`
	} `yaml:"fields"`
	Views []struct {
		Name string `yaml:"name"`
	} `yaml:"views"`
	Workflows []struct {
		Name           string `yaml:"name"`
		Number         int    `yaml:"number"`
		Enabled        bool   `yaml:"enabled"`
		ExpectedAction string `yaml:"expected_action"`
		Required       bool   `yaml:"required"`
	} `yaml:"workflows"`
}

// LoadDefinition parses the embedded board.yml.
func LoadDefinition() (*Definition, error) {
	var d Definition
	if err := yaml.Unmarshal(definitionYAML, &d); err != nil {
		return nil, fmt.Errorf("parsing embedded board.yml: %w", err)
	}
	return &d, nil
}

// StatusOptionNames returns the Status option names in column order.
func (d *Definition) StatusOptionNames() []string {
	names := make([]string, len(d.Fields.Status.Options))
	for i, o := range d.Fields.Status.Options {
		names[i] = o.Name
	}
	return names
}

// ViewState is what the live Board reports.
type ViewState struct {
	StatusOptions []string        // option names in order; nil if no Status field
	Views         []string        // view names
	Workflows     map[string]bool // workflow name -> enabled
}

// SettingsURL is where a Board's workflows are configured in the UI.
func SettingsURL(ownerType, owner string, number int) string {
	kind := "users"
	if ownerType == "Organization" {
		kind = "orgs"
	}
	return fmt.Sprintf("https://github.com/%s/%s/projects/%d/workflows", kind, owner, number)
}

// Check compares a live Board against the definition and returns one
// human-readable problem (with the manual fix) per mismatch. workflowsURL is
// where the user fixes workflows; requiredOnly limits the workflow check to
// the ones the tool depends on.
func (d *Definition) Check(live ViewState, workflowsURL string, requiredOnly bool) []string {
	var problems []string

	if !requiredOnly {
		want := d.StatusOptionNames()
		switch {
		case live.StatusOptions == nil:
			problems = append(problems, fmt.Sprintf("the %s field is missing", d.Fields.Status.Name))
		case strings.Join(live.StatusOptions, "|") != strings.Join(want, "|"):
			problems = append(problems, fmt.Sprintf(
				"the %s field has options [%s], want exactly [%s] in this order (Project settings > Fields > %s; do not delete the field, that detaches every card)",
				d.Fields.Status.Name, strings.Join(live.StatusOptions, ", "), strings.Join(want, ", "), d.Fields.Status.Name))
		}

		for _, v := range d.Views {
			found := false
			for _, name := range live.Views {
				found = found || name == v.Name
			}
			if !found {
				problems = append(problems, fmt.Sprintf("the %q view is missing: create a board-layout view grouped by Status", v.Name))
			}
		}
	}

	for _, w := range d.Workflows {
		if requiredOnly && !w.Required {
			continue
		}
		enabled, known := live.Workflows[w.Name]
		switch {
		case !known:
			problems = append(problems, fmt.Sprintf(
				"the %q workflow was not found on the board. Open %s and check it exists (the token also needs the `project` scope: `gh auth refresh -s project`)",
				w.Name, workflowsURL))
		case enabled != w.Enabled:
			problems = append(problems, workflowFix(w.Name, w.Enabled, w.ExpectedAction, workflowsURL))
		}
	}
	return problems
}

// workflowFix explains how to put one workflow in the wanted state. GitHub
// has no API to enable or configure project workflows, so this is manual.
func workflowFix(name string, wantEnabled bool, action, url string) string {
	if !wantEnabled {
		return fmt.Sprintf("the %q workflow must be OFF (it would race the tool's first Status). Open %s, select %q and turn it off", name, url, name)
	}
	msg := fmt.Sprintf("the %q workflow must be ON. Open %s, select %q", name, url, name)
	if action != "" && !strings.HasPrefix(action, "(") {
		msg += fmt.Sprintf(", set its action to %q", action)
	}
	return msg + " and turn it on (Save and turn on workflow)"
}
