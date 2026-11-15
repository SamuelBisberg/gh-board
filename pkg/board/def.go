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

// Check compares a live Board against the definition and returns one
// human-readable problem (with the manual fix) per mismatch.
func (d *Definition) Check(live ViewState) []string {
	var problems []string

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
			problems = append(problems, fmt.Sprintf("the %q view is missing: create a board-layout view grouped by Status (views cannot be created through the API)", v.Name))
		}
	}

	for _, w := range d.Workflows {
		enabled, known := live.Workflows[w.Name]
		if !known || enabled == w.Enabled {
			continue
		}
		state := map[bool]string{true: "enabled", false: "disabled"}
		problems = append(problems, fmt.Sprintf("the %q workflow is %s, want it %s (Project settings > Workflows)", w.Name, state[enabled], state[w.Enabled]))
	}
	return problems
}
