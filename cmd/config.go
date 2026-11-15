package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-board/pkg/config"
	"github.com/SamuelBisberg/gh-board/pkg/git"
	"github.com/SamuelBisberg/gh-board/pkg/tmpl"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "View or change gh-board settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList()
		},
	}
	c.AddCommand(newConfigListCmd(), newConfigGetCmd(), newConfigSetCmd(), newConfigTemplatesCmd())
	return c
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show every setting and where it's stored",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList()
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print the value of a single setting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(args[0])
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Change a setting and save it",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(args[0], args[1])
		},
	}
}

func runConfigList() error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}

	path, err := config.Path()
	if err == nil {
		theme.Infof("Config file: %s", path)
	}
	for _, key := range append(config.Keys(), "team") {
		value, _ := cfg.Get(key)
		fmt.Printf("  %s = %s\n", key, value)
	}

	return nil
}

// runConfigGet prints a single raw value with no styling, since it's meant
// to be consumed by scripts (e.g. `gh board config get prefix`).
func runConfigGet(key string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	value, err := cfg.Get(key)
	if err != nil {
		return err
	}
	fmt.Println(value)
	return nil
}

func runConfigSet(key, value string) error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	if err := cfg.Set(key, value); err != nil {
		return err
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	theme.Successf("Set %s = %s", key, value)
	return nil
}

func newConfigTemplatesCmd() *cobra.Command {
	var initFiles bool
	c := &cobra.Command{
		Use:   "templates",
		Short: "Show where each template comes from (--init writes the defaults)",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runConfigTemplates(initFiles) },
	}
	c.Flags().BoolVar(&initFiles, "init", false, "write the embedded defaults into the override directory")
	return c
}

func runConfigTemplates(initFiles bool) error {
	_, theme, err := loadTheme()
	if err != nil {
		return err
	}
	dir, err := config.TemplatesDir()
	if err != nil {
		return err
	}
	if initFiles {
		written, err := tmpl.WriteDefaults(dir)
		if err != nil {
			return fmt.Errorf("writing templates: %w", err)
		}
		for _, w := range written {
			theme.Successf("Wrote %s", w)
		}
		if len(written) == 0 {
			theme.Infof("Templates already exist in %s", dir)
		}
		return nil
	}
	root, _ := git.RepoRoot()
	for _, k := range []tmpl.Kind{tmpl.Issue, tmpl.PR} {
		_, src := tmpl.Resolve(k, root, dir)
		fmt.Printf("%-6s %s\n", k, src)
	}
	return nil
}
