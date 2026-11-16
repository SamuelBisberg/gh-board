package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/board"
	"github.com/SamuelBisberg/gh-board/pkg/gh"
)

func newBrowseCmd() *cobra.Command {
	var print bool
	c := &cobra.Command{
		Use:   "browse",
		Short: "Open the board on github.com",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runBrowse(print) },
	}
	c.Flags().BoolVarP(&print, "print", "p", false, "print the URL instead of opening it")
	return c
}

func runBrowse(printOnly bool) error {
	if err := checkAuth(); err != nil {
		return err
	}
	cfg, _, err := loadTheme()
	if err != nil {
		return err
	}
	if cfg.Board.Owner == "" || cfg.Board.Project == 0 {
		return fmt.Errorf("no board configured: run `gh board init`")
	}
	ownerType, _ := gh.Read("api", "users/"+cfg.Board.Owner, "--jq", ".type")
	url := board.URL(ownerType, cfg.Board.Owner, cfg.Board.Project)
	if printOnly {
		fmt.Println(url)
		return nil
	}
	return openURL(url)
}
