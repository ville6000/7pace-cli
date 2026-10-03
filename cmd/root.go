// Package cmd implements the 7pace-cli commands.
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/7pace-cli/internal/config"
)

// Execute runs the command line with ctx, which every command passes on to its
// API requests. Cobra has already printed the returned error.
func Execute(ctx context.Context) error {
	return NewRootCmd().ExecuteContext(ctx)
}

// NewRootCmd builds the 7pace-cli command tree. Each call has its own
// configuration and flags, so a tree can be built and run independently of
// any other.
func NewRootCmd() *cobra.Command {
	return newRootCmd(viper.New())
}

// newRootCmd builds the command tree around v, which holds the configuration
// every command reads and writes. Tests pass their own v to preset values.
func newRootCmd(v *viper.Viper) *cobra.Command {
	var cfgFile string

	cmd := &cobra.Command{
		Use:     "7pace-cli",
		Short:   "Post worklogs to an on-prem 7pace Timetracker",
		Long:    "Post worklogs to an on-prem 7pace Timetracker, one at a time or from toggl-cli time entries.",
		Version: buildVersion(),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return loadConfig(cmd, v, cfgFile)
		},
	}

	cmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"config file (default is $XDG_CONFIG_HOME/7pace-cli/config.yaml or ~/.config/7pace-cli/config.yaml)",
	)

	cmd.AddCommand(
		newAddCmd(v),
		newConfigCmd(v),
		newSyncCmd(v),
	)

	return cmd
}

// loadConfig reads the config file into v: cfgFile when given, otherwise the
// default location. A missing file is not an error; commands report the
// settings they need.
func loadConfig(cmd *cobra.Command, v *viper.Viper, cfgFile string) error {
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		configPath, err := ConfigPath()
		if err != nil {
			return err
		}
		v.SetConfigFile(configPath)
	}

	config.UseEnv(v)

	if err := v.ReadInConfig(); err == nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Using config file:", v.ConfigFileUsed())
	}

	return nil
}
