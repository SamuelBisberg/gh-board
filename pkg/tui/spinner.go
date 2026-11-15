package tui

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

var spinnerFrames = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

// spinnerDelay keeps fast calls from flashing a spinner.
const spinnerDelay = 150 * time.Millisecond

// IsInteractive reports whether stdout is a terminal, i.e. whether
// interactive prompts and lists can be shown at all.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// RunWithSpinner shows an inline spinner labeled label while fn runs, but
// only if fn takes longer than a moment. It draws on stderr and only when
// stderr is a terminal, so piped or scripted output (a printed issue URL,
// `config get`) stays clean.
func (t *Theme) RunWithSpinner(label string, fn func() error) error {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return fn()
	}

	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-done:
			return
		case <-time.After(spinnerDelay):
		}
		defer fmt.Fprint(os.Stderr, "\r\033[K")
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for i := 0; ; i++ {
			fmt.Fprintf(os.Stderr, "\r%s %s", t.Info.Render(string(spinnerFrames[i%len(spinnerFrames)])), label)
			select {
			case <-done:
				return
			case <-ticker.C:
			}
		}
	}()

	defer func() {
		close(done)
		<-stopped
	}()
	return fn()
}
