package board

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/SamuelBisberg/gh-board/pkg/gh"
)

// Board is a handle on the GitHub Project, with the IDs every command
// needs resolved once per run.
type Board struct {
	Owner         string
	Number        int
	ProjectID     string
	StatusFieldID string
	optionIDs     map[Status]string
}

// Item is one Card on the Board.
type Item struct {
	ID     string
	Type   string // "Issue" or "PullRequest"
	Number int
	Repo   string // owner/repo
	URL    string
	Title  string
	Status Status
}

// Open resolves the project, its Status field and the six option IDs.
func Open(owner string, number int) (*Board, error) {
	n := strconv.Itoa(number)
	var project struct {
		ID string `json:"id"`
	}
	if err := gh.ReadJSON(&project, "project", "view", n, "--owner", owner, "--format", "json"); err != nil {
		return nil, fmt.Errorf("opening board %s #%d: %w", owner, number, err)
	}

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
		return nil, fmt.Errorf("reading board fields: %w", err)
	}

	b := &Board{Owner: owner, Number: number, ProjectID: project.ID, optionIDs: map[Status]string{}}
	for _, f := range fields.Fields {
		if f.Name != "Status" {
			continue
		}
		b.StatusFieldID = f.ID
		for _, o := range f.Options {
			b.optionIDs[Status(o.Name)] = o.ID
		}
	}
	if b.StatusFieldID == "" {
		return nil, fmt.Errorf("board %s #%d has no Status field - run `gh board init`", owner, number)
	}
	for _, s := range All {
		if b.optionIDs[s] == "" {
			return nil, fmt.Errorf("board %s #%d has no %q Status option - run `gh board init`", owner, number, s)
		}
	}
	return b, nil
}

// Items lists every Card on the Board.
func (b *Board) Items() ([]Item, error) {
	var raw struct {
		Items []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Status  string `json:"status"`
			Content struct {
				Type       string `json:"type"`
				Number     int    `json:"number"`
				Title      string `json:"title"`
				URL        string `json:"url"`
				Repository string `json:"repository"`
			} `json:"content"`
		} `json:"items"`
	}
	if err := gh.ReadJSON(&raw, "project", "item-list", strconv.Itoa(b.Number), "--owner", b.Owner, "--limit", "1000", "--format", "json"); err != nil {
		return nil, fmt.Errorf("listing board items: %w", err)
	}
	items := make([]Item, 0, len(raw.Items))
	for _, r := range raw.Items {
		items = append(items, Item{
			ID: r.ID, Type: r.Content.Type, Number: r.Content.Number, Repo: r.Content.Repository,
			URL: r.Content.URL, Title: r.Content.Title, Status: Status(r.Status),
		})
	}
	return items, nil
}

// Find returns the Card for a PR or issue URL.
func Find(items []Item, url string) (Item, bool) {
	for _, it := range items {
		if it.URL == url {
			return it, true
		}
	}
	return Item{}, false
}

// Add puts an issue or PR on the Board and returns its Card ID. Adding
// something already on the Board returns the existing Card.
func (b *Board) Add(url string) (string, error) {
	out, err := gh.Write("project", "item-add", strconv.Itoa(b.Number), "--owner", b.Owner, "--url", url, "--format", "json")
	if err != nil {
		return "", fmt.Errorf("adding %s to the board: %w", url, err)
	}
	if out == "" {
		return "", nil
	}
	var r struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return "", fmt.Errorf("decoding item-add output: %w", err)
	}
	return r.ID, nil
}

// SetStatus moves a Card. The tool never writes Closed.
func (b *Board) SetStatus(itemID string, s Status) error {
	if s == Closed {
		return fmt.Errorf("refusing to write Closed: the board's own workflow owns that column")
	}
	opt, ok := b.optionIDs[s]
	if !ok {
		return fmt.Errorf("unknown status %q", s)
	}
	_, err := gh.Write("project", "item-edit", "--id", itemID, "--project-id", b.ProjectID,
		"--field-id", b.StatusFieldID, "--single-select-option-id", opt)
	if err != nil {
		return fmt.Errorf("setting status %q: %w", s, err)
	}
	return nil
}

// Track adds url to the Board (idempotently) and sets its Status.
func (b *Board) Track(url string, s Status) error {
	id, err := b.Add(url)
	if err != nil {
		return err
	}
	return b.SetStatus(id, s)
}

// Remove deletes a Card from the Board.
func (b *Board) Remove(itemID string) error {
	_, err := gh.Write("project", "item-delete", strconv.Itoa(b.Number), "--owner", b.Owner, "--id", itemID)
	if err != nil {
		return fmt.Errorf("removing card: %w", err)
	}
	return nil
}
