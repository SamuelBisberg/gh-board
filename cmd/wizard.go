package cmd

import (
	"fmt"
	"strings"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/browser"
	"github.com/SamuelBisberg/gh-board/pkg/tui"
)

// Replaced in tests.
var (
	openURL = func(u string) error { return browser.Open(u) }
	pickOne = tui.PickOne
	confirm = tui.Confirm
)

// workflowInstructions says what to click on one workflow's page.
func workflowInstructions(s board.WorkflowStep) []string {
	if !s.Enable {
		return []string{"Switch the \"Enabled\" toggle OFF."}
	}
	var lines []string
	if status, ok := strings.CutPrefix(s.Action, "Set Status = "); ok {
		lines = append(lines, fmt.Sprintf("Under \"Set\", choose Status = %q.", status))
	}
	return append(lines, "Click \"Save and turn on workflow\" (or switch \"Enabled\" ON).")
}

// guideWorkflows walks the user through the workflows GitHub gives no API
// for: each page is opened, the clicks are spelled out, and the live board
// is re-read to confirm the step took. It returns once every step is done
// or skipped; skipped steps are reported, never fatal.
func guideWorkflows(theme *tui.Theme, steps []board.WorkflowStep, owner string, number int, reread func() (board.ViewState, error)) error {
	ok, err := confirm(fmt.Sprintf("%d board workflow(s) need flipping by hand (GitHub has no API for it). Walk through them now? Your browser will open each page.", len(steps)))
	if err != nil || !ok {
		return err
	}
	var skipped []string
	for i, s := range steps {
		fmt.Printf("\n%s\n", theme.Header(fmt.Sprintf("Workflow %d of %d: %s", i+1, len(steps), s.Name)))
		if s.Reason != "" {
			fmt.Println("  " + strings.Join(strings.Fields(s.Reason), " "))
		}
		if err := openURL(s.URL); err != nil {
			theme.Warningf("couldn't open the browser; open it yourself: %s", s.URL)
		} else {
			theme.Infof("Opened %s", s.URL)
		}
		for n, line := range workflowInstructions(s) {
			fmt.Printf("  %d. %s\n", n+1, line)
		}
		done := false
		for !done {
			idx, err := pickOne("When you're done", []string{"Done: check it", "Open the page again", "Skip this one"})
			if err != nil {
				return err
			}
			switch idx {
			case 1:
				_ = openURL(s.URL)
			case 2:
				skipped = append(skipped, s.Name)
				done = true
			default:
				live, err := reread()
				if err != nil {
					return fmt.Errorf("re-reading board %s #%d: %w", owner, number, err)
				}
				if on, known := live.Workflows[s.Name]; known && on == s.Enable {
					theme.Successf("%q is %s", s.Name, onOff(s.Enable))
					done = true
				} else {
					theme.Warningf("GitHub still shows %q as %s. Did the change save?", s.Name, onOff(on))
				}
			}
		}
	}
	if len(skipped) > 0 {
		theme.Warningf("Skipped: %s. Optional: `gh board sync` still derives every Status.", strings.Join(skipped, ", "))
	} else {
		theme.Successf("All workflows are set as expected")
	}
	return nil
}

func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}
