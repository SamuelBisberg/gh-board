package main

import (
	"os"

	"github.com/SamuelBisberg/gh-board/cmd"
	"github.com/SamuelBisberg/gh-board/pkg/tui"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z"
// (cli/gh-extension-precompile does this automatically on release).
var version = "dev"

func main() {
	cmd.Version = version

	if err := cmd.NewRootCmd().Execute(); err != nil {
		tui.New().Errorf("%v", err)
		os.Exit(1)
	}
}
