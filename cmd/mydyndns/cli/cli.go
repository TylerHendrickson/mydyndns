// Package cli provides a CLI application that can be used to interact with a remote MyDynDNS web service.
package cli

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultConfigPath     = "."
	defaultConfigFilename = "mydyndns"
	envPrefix             = "MYDYNDNS"
)

var (
	Version             = "dev"
	defaultPollInterval = time.Hour
	minimumPollInterval = time.Second * 10
)

// Execute runs the mydyndns CLI application
func Execute() error {
	return newCLI().Execute()
}

// ExecuteContext runs the mydyndns CLI application with a Context
func ExecuteContext(ctx context.Context) error {
	return newCLI().ExecuteContext(ctx)
}

// newCLI creates and returns a new *cobra.Command "root" command, assembling child/sub commands
// with the following nested hierarchy (note this does not include Cobra-provided subcommands
// such as "completion" or "help"):
//   mydyndns
//   ├── agent
//   │   └── start
//   ├── api
//   │   ├── my-ip
//   │   └── update-alias
//   └── config
//       ├── show
//       ├── types
//       │   ├── check
//       │   └── list
//       ├── validate
//       └── write
func newCLI() *cobra.Command {
	// mydyndns ...
	rootCmd := newRootCmd()

	// mydyndns api ...
	apiCmd := newAPICmd()
	apiCmd.AddCommand(newAPIMyIPCmd(), newAPIUpdateAliasCmd())
	rootCmd.AddCommand(apiCmd)

	// mydyndns agent ...
	agentCmd := newAgentCmd()
	agentCmd.AddCommand(newAgentStartCmd())
	rootCmd.AddCommand(agentCmd)

	// mydyndns config ...
	configCmd := newConfigCmd()
	configCmd.AddCommand(newConfigWriteCmd(), newConfigShowCmd(), newConfigValidateCmd())
	rootCmd.AddCommand(configCmd)

	// mydyndns config types ...
	configTypesCmd := newConfigTypesCmd()
	configTypesCmd.AddCommand(newConfigTypesCheckCmd(), newConfigTypesListCmd())
	configCmd.AddCommand(configTypesCmd)

	// (HIDDEN) mydyndns command-tree ...
	rootCmd.AddCommand(newCommandTreeCmd())

	return rootCmd
}
