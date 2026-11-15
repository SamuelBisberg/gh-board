// Package progress lets the low-level gh and git wrappers report slow I/O
// without depending on the terminal UI: the command layer installs a
// spinner, everything else just runs the function.
package progress

// Run runs fn while showing label. The default shows nothing.
var Run = func(label string, fn func() error) error { return fn() }
