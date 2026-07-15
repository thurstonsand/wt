package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/thurstonsand/wt/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage wt configuration",
		Long: `Manage wt configuration stored at ~/.wt/config.yaml.

Subcommands:
  show               Show all config values with descriptions
  get <key>          Get a specific config value
  set <key> <value>  Set a config value (replaces list values entirely)
  add <key> <value>  Add item to a list config value
  remove <key> [index]  Remove item from a list config value
  edit               Open config in $EDITOR`,
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigAddCmd())
	cmd.AddCommand(newConfigRemoveCmd())
	cmd.AddCommand(newConfigEditCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show all config values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store := config.DefaultGlobalStore()
			cfg, isNew, err := store.LoadConfig()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if isNew {
				_, _ = fmt.Fprintln(out, "Created default config at", store.ConfigPath())
				_, _ = fmt.Fprintln(out)
			}

			w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
			for i, field := range config.ConfigFields {
				field.FprintValue(w, cfg)
				if i < len(config.ConfigFields)-1 {
					_, _ = fmt.Fprintln(w)
				}
			}
			return w.Flush()
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "get <key>",
		Short:     "Get a config value",
		ValidArgs: config.ValidKeys(),
		Args:      cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			field := config.GetFieldMeta(key)
			if field == nil {
				return config.UnknownKeyError(key)
			}

			store := config.DefaultGlobalStore()
			cfg, _, err := store.LoadConfig()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 1, ' ', 0)
			field.FprintValue(w, cfg)
			return w.Flush()
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "set <key> <value>",
		Short:     "Set a config value",
		ValidArgs: config.ValidKeys(),
		Args:      cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			store := config.DefaultGlobalStore()
			if err := store.SetConfigValueFromString(key, value); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", key, value)
			return nil
		},
	}
}

func newConfigAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "add <key> <value>",
		Short:     "Add item to a list config value",
		ValidArgs: config.ListKeys(),
		Args:      cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			store := config.DefaultGlobalStore()
			if err := store.AddToConfigList(key, value); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %q to %s\n", value, key)
			return nil
		},
	}
}

func newConfigRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <key> [index|value]",
		Short: "Remove item from a list config value",
		Long: `Remove an item from a list config value by 0-based index or exact match.

If no value is given, the current entries are printed with indices.
If the value is a number and a valid index, the entry at that index is removed.
Otherwise, the first exact match is removed.`,
		ValidArgs: config.ListKeys(),
		Args:      cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			store := config.DefaultGlobalStore()

			if len(args) == 1 {
				return configRemoveShowEntries(cmd, store, key)
			}

			value := args[1]
			return configRemoveEntry(cmd, store, key, value)
		},
	}
}

func configRemoveShowEntries(cmd *cobra.Command, store *config.GlobalStore, key string) error {
	field := config.GetFieldMeta(key)
	if field == nil {
		return config.UnknownKeyError(key)
	}
	if !field.IsList {
		return fmt.Errorf("remove requires a value for non-list field %q", key)
	}

	cfg, _, err := store.LoadConfig()
	if err != nil {
		return err
	}

	entries := cfg.GetListValue(key)
	out := cmd.OutOrStdout()
	if len(entries) == 0 {
		_, _ = fmt.Fprintf(out, "%s is empty\n", key)
		return nil
	}

	_, _ = fmt.Fprintf(out, "%s entries:\n", key)
	for i, entry := range entries {
		_, _ = fmt.Fprintf(out, "  [%d] %s\n", i, entry)
	}
	_, _ = fmt.Fprintf(out, "\nUse: wt config remove %s <index>\n", key)
	return nil
}

func configRemoveEntry(cmd *cobra.Command, store *config.GlobalStore, key, value string) error {
	// Try index-based removal first
	if idx, err := strconv.Atoi(value); err == nil {
		removed, err := store.RemoveFromConfigListByIndex(key, idx)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed [%d] %q from %s\n", idx, removed, key)
		return nil
	}

	// Fall back to exact string match
	removed, err := store.RemoveFromConfigList(key, value)
	if err != nil {
		return err
	}
	if removed == "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No match for %q in %s\n", value, key)
		return nil
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed %q from %s\n", removed, key)
	return nil
}

func newConfigEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open config in editor",
		Long:  "Open config in $EDITOR (or vi if unset) and wait for it to close.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			store := config.DefaultGlobalStore()

			// Ensure config exists
			_, _, err := store.LoadConfig()
			if err != nil {
				return err
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}

			path := store.ConfigPath()
			parts := strings.Fields(editor)
			parts = append(parts, path)
			//nolint:gosec // editor is from trusted env or default
			c := exec.Command(parts[0], parts[1:]...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr

			if err := c.Run(); err != nil {
				return fmt.Errorf("editor exited with error: %w", err)
			}
			return nil
		},
	}
}
