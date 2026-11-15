package board

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/SamuelBisberg/gh-board/pkg/gh"
)

// graphqlString quotes s as a GraphQL string literal.
func graphqlString(s string) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(s)
	return strings.TrimSpace(b.String())
}

// statusMutation builds the updateProjectV2Field mutation that sets the
// Status options to the definition's, in order. Options that already exist
// keep their ID (matched by name), so cards stay attached to their Status.
func (d *Definition) statusMutation(fieldID string, existing map[string]string) string {
	var opts []string
	for _, o := range d.Fields.Status.Options {
		parts := []string{"name:" + graphqlString(o.Name), "color:" + o.Color, "description:" + graphqlString(o.Description)}
		if id := existing[o.Name]; id != "" {
			parts = append([]string{"id:" + graphqlString(id)}, parts...)
		}
		opts = append(opts, "{"+strings.Join(parts, ",")+"}")
	}
	return fmt.Sprintf(`mutation{updateProjectV2Field(input:{fieldId:%s,singleSelectOptions:[%s]}){projectV2Field{... on ProjectV2SingleSelectField{id}}}}`,
		graphqlString(fieldID), strings.Join(opts, ","))
}

// Setup makes the Board match the definition: it creates a missing Status
// field, rewrites a Status field whose options differ (keeping the IDs of
// options that already exist), and creates missing views. It returns what
// it did, one line per change.
func (d *Definition) Setup(owner string, number int, live ViewState) ([]string, error) {
	var done []string
	n := strconv.Itoa(number)
	want := strings.Join(d.StatusOptionNames(), "|")

	switch {
	case live.StatusOptions == nil:
		if _, err := gh.Write("project", "field-create", n, "--owner", owner, "--name", d.Fields.Status.Name,
			"--data-type", "SINGLE_SELECT", "--single-select-options", strings.Join(d.StatusOptionNames(), ",")); err != nil {
			return done, fmt.Errorf("creating the Status field: %w", err)
		}
		done = append(done, "created the Status field")

	case strings.Join(live.StatusOptions, "|") != want:
		var fields struct {
			Fields []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Options []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"options"`
			} `json:"fields"`
		}
		if err := gh.ReadJSON(&fields, "project", "field-list", n, "--owner", owner, "--format", "json"); err != nil {
			return done, fmt.Errorf("reading the Status field: %w", err)
		}
		fieldID, existing := "", map[string]string{}
		for _, f := range fields.Fields {
			if f.Name == d.Fields.Status.Name {
				fieldID = f.ID
				for _, o := range f.Options {
					existing[o.Name] = o.ID
				}
			}
		}
		if fieldID == "" {
			return done, fmt.Errorf("the Status field disappeared while updating it")
		}
		if _, err := gh.Write("api", "graphql", "-f", "query="+d.statusMutation(fieldID, existing)); err != nil {
			return done, fmt.Errorf("updating the Status options: %w", err)
		}
		done = append(done, fmt.Sprintf("set the Status options to [%s] (was [%s])",
			strings.Join(d.StatusOptionNames(), ", "), strings.Join(live.StatusOptions, ", ")))
	}

	for _, v := range d.Views {
		found := false
		for _, name := range live.Views {
			found = found || name == v.Name
		}
		if found {
			continue
		}
		q := fmt.Sprintf(`mutation{createProjectV2View(input:{projectId:%s,name:%s,layout:BOARD_LAYOUT}){projectV2View{id}}}`,
			graphqlString(live.ProjectID), graphqlString(v.Name))
		if _, err := gh.Write("api", "graphql", "-f", "query="+q); err != nil {
			return done, fmt.Errorf("creating the %q view: %w", v.Name, err)
		}
		done = append(done, fmt.Sprintf("created the %q board view", v.Name))
	}
	return done, nil
}
