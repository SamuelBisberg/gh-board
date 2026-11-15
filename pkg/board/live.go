package board

import (
	"strconv"

	"github.com/SamuelBisberg/gh-board/pkg/gh"
)

// ViewState is what the live Board reports.
type ViewState struct {
	ProjectID     string
	StatusOptions []string        // option names in order; nil if no Status field
	Views         []string        // view names
	Workflows     map[string]bool // workflow name -> enabled
}

type liveBoard struct {
	ID     string `json:"id"`
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

const liveQuery = `query($o:String!,$n:Int!){repositoryOwner(login:$o){... on ProjectV2Owner{projectV2(number:$n){id ` +
	`fields(first:50){nodes{... on ProjectV2SingleSelectField{name options{name}}}}` +
	`views(first:20){nodes{name}} workflows(first:30){nodes{name enabled}}}}}}`

// ReadLive reads the live Board: its Status options, views and workflows.
func ReadLive(owner string, number int) (ViewState, error) {
	var r struct {
		Data struct {
			Owner struct {
				ProjectV2 liveBoard `json:"projectV2"`
			} `json:"repositoryOwner"`
		} `json:"data"`
	}
	if err := gh.ReadJSON(&r, "api", "graphql", "-f", "query="+liveQuery, "-f", "o="+owner, "-F", "n="+strconv.Itoa(number)); err != nil {
		return ViewState{}, err
	}
	p := r.Data.Owner.ProjectV2
	st := ViewState{ProjectID: p.ID, Workflows: map[string]bool{}}
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
