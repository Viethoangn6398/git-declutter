package cmd

import (
	"fmt"
	"strings"

	"github.com/kunmi02/git-declutter/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "View or change GitDeclutter configuration",
	}
	c.AddCommand(newConfigListCmd(), newConfigGetCmd(), newConfigSetCmd(), newConfigAddCmd())
	return c
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Print the effective configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Fprint(stdout(), string(data))
			path, _ := config.GlobalPath()
			fmt.Fprintf(stdout(), "\n# %s\n", path)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			v, err := getKey(cfg, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(stdout(), v)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := setKey(&cfg, args[0], args[1]); err != nil {
				return err
			}
			return config.Save(cfg)
		},
	}
}

func newConfigAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add protected <pattern>",
		Short: "Add a protected branch pattern",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "protected" {
				return fmt.Errorf("unknown add target %q (expected protected)", args[0])
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.Protected = append(cfg.Protected, args[1])
			return config.Save(cfg)
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the GitDeclutter version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(stdout(), "git-declutter %s\n", appVersion)
			return nil
		},
	}
}

func getKey(cfg config.Config, key string) (string, error) {
	switch strings.ToLower(key) {
	case "recovery.retention":
		return cfg.Recovery.Retention, nil
	case "recovery.enabled":
		return fmt.Sprintf("%t", cfg.Recovery.Enabled), nil
	case "cleanup.requireremotedeleted":
		return fmt.Sprintf("%t", cfg.Cleanup.RequireRemoteDeleted), nil
	default:
		return "", fmt.Errorf("unknown key %q", key)
	}
}

func setKey(cfg *config.Config, key, value string) error {
	switch strings.ToLower(key) {
	case "recovery.retention":
		if _, forever, err := (config.Config{Recovery: config.RecoveryConfig{Retention: value}}).RetentionDuration(); err != nil && !forever {
			if _, err := config.ParseDuration(value); err != nil && !strings.EqualFold(value, "forever") {
				return fmt.Errorf("invalid retention %q", value)
			}
		}
		cfg.Recovery.Retention = value
		return nil
	case "recovery.enabled":
		cfg.Recovery.Enabled = parseBool(value)
		return nil
	case "cleanup.requireremotedeleted", "cleanupRemoteExistingBranches":
		if strings.EqualFold(key, "cleanupRemoteExistingBranches") {
			cfg.Cleanup.RequireRemoteDeleted = !parseBool(value)
			return nil
		}
		cfg.Cleanup.RequireRemoteDeleted = parseBool(value)
		return nil
	default:
		return fmt.Errorf("unknown key %q", key)
	}
}

func parseBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
