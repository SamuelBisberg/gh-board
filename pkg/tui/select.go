package tui

import (
	"errors"

	"github.com/charmbracelet/huh"
)

// PickMany shows a multi-select (space toggles, enter confirms) and returns
// the chosen values. Esc skips and returns nothing.
func PickMany(title string, options []string) ([]string, error) {
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	var chosen []string
	err := huh.NewMultiSelect[string]().Title(title).Options(opts...).Value(&chosen).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return nil, nil
	}
	return chosen, err
}

// PickOne shows a single select and returns the chosen option's index.
func PickOne(title string, options []string) (int, error) {
	opts := make([]huh.Option[int], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, i)
	}
	var idx int
	err := huh.NewSelect[int]().Title(title).Options(opts...).Value(&idx).Run()
	return idx, err
}

// Confirm asks a yes/no question.
func Confirm(title string) (bool, error) {
	var ok bool
	err := huh.NewConfirm().Title(title).Value(&ok).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return false, nil
	}
	return ok, err
}
