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
				Name        string `yaml:"name"`
				Color       string `yaml:"color"`
				Description string `yaml:"description"`
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

// SettingsURL is where a Board's workflows are configured in the UI.
func SettingsURL(ownerType, owner string, number int) string {
	kind := "users"
	if ownerType == "Organization" {
		kind = "orgs"
	}
	return fmt.Sprintf("https://github.com/%s/%s/projects/%d/workflows", kind, owner, number)
}

// WorkflowAdvice lists the project workflows the tool would like set a
// certain way. GitHub has no API to enable, disable or configure them, so
// these can only be advice: every Status is re-derived from the PR by
// `sync` anyway, the workflows just make cards move a moment sooner.
// Only workflows marked required are covered.
func (d *Definition) WorkflowAdvice(live ViewState, workflowsURL string) []string {
	var advice []string
	for _, w := range d.Workflows {
		if !w.Required {
			continue
		}
		enabled, known := live.Workflows[w.Name]
		if known && enabled == w.Enabled {
			continue
		}
		advice = append(advice, workflowFix(w.Name, w.Enabled, w.ExpectedAction, workflowsURL))
	}
	return advice
}

// workflowFix explains how to put one workflow in the wanted state.
func workflowFix(name string, wantEnabled bool, action, url string) string {
	if !wantEnabled {
		return fmt.Sprintf("turn OFF %q (it can overwrite the Status the tool just set): %s", name, url)
	}
	msg := fmt.Sprintf("turn ON %q", name)
	if action != "" && !strings.HasPrefix(action, "(") {
		msg += fmt.Sprintf(" with action %q", action)
	}
	return msg + ": " + url
}
