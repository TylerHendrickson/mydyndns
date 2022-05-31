package cli

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/TylerHendrickson/mydyndns/pkg/sdk"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Version: Version,
		Use:     "mydyndns",
		Short:   "Dynamic DNS utility",
		Long: `mydyndns is a dynamic DNS utility. It offers a configurable agent which can be used to periodically
refresh from and send updates to a remote DNS management service.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := bootstrapConfig(cmd); err != nil {
				return err
			}
			return bootstrapAPIClient(cmd)
		},
	}

	// Outputs
	cmd.SetOut(cmd.OutOrStdout())
	cmd.SetErr(cmd.OutOrStderr())

	// Global flags
	cmd.PersistentFlags().String("config-file", "",
		"Explicitly set a config file (disables config file discovery)")
	cmd.PersistentFlags().String("config-path", defaultConfigPath,
		"Search path for config file discovery when --config-file is not set to an absolute path.")

	cmd.PersistentFlags().StringP("api-url", "u", "",
		"Base URL for the mydyndns control API")
	cmd.PersistentFlags().DurationP("interval", "i", defaultPollInterval,
		"How often to poll for a new IP")
	cmd.PersistentFlags().StringP("api-key", "k", "",
		"Client API secret")
	cmd.PersistentFlags().CountP("log-verbosity", "v",
		"Increase logging verbosity level (default ERROR)")
	cmd.PersistentFlags().Bool("log-json", false,
		"Whether to output JSON logs")

	return cmd
}

func bootstrapConfig(cmd *cobra.Command) error {
	// Matching environment variables must have prefix MYDYNDNS_
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	// Explicitly bind config-path and config-file flag/env var directives
	bugIfError(viper.BindPFlag("config-path", cmd.Flag("config-path")), "could not bootstrap config")
	bugIfError(viper.BindPFlag("config-file", cmd.Flag("config-file")), "could not bootstrap config")
	_ = viper.BindEnv("config-path", flagNameToEnvVar(envPrefix, "config-path"))
	_ = viper.BindEnv("config-file", flagNameToEnvVar(envPrefix, "config-file"))
	_ = viper.BindPFlags(cmd.Flags())

	if viper.IsSet("config-file") {
		configFilename := viper.GetString("config-file")
		if !filepath.IsAbs(configFilename) {
			configFilename = filepath.Join(viper.GetString("config-path"), configFilename)
		}
		viper.SetConfigFile(configFilename)
	} else {
		viper.SetConfigName(defaultConfigFilename)
		viper.AddConfigPath(viper.GetString("config-path"))
	}

	if err := func() (e error) {
		// Because not all underlying errors are graceful (the TOML parser seems fragile),
		// attempt to recover from a parsing-related panic as gracefully as possible
		defer func() {
			if r := recover(); r != nil {
				cmd.SilenceUsage = true
				e = fmt.Errorf(
					"unrecoverable error reading (possibly corrupt) config file %q due to underlying error: %q",
					viper.ConfigFileUsed(), r,
				)
			}
		}()

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok || viper.IsSet("config-file") {
				return err
			}
		}
		return nil
	}(); err != nil {
		return err
	}

	// TODO: We do this because command handlers generally retrieve all config directives from `cmd.Flags()`.
	//  Effectively, this is just syncing viper directives with (back to) their respective flag values.
	// 	If things were refactored to use Viper for fetching canonical/resolved values, we could get rid of this.
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && viper.IsSet(f.Name) {
			// FIXME: Viper now uses TOMLv2, which does not reliably format durations as a string,
			// and breaks cmd.Flags().Set() with underlying error: "time: missing unit in duration".
			// It would be nice if marshalling time.Duration flag values to TOML were reliably
			// formatted in human-readable form (e.g. "1h23m45s"), but I think this exposes the
			// brittleness of what we're doing here; we really should just use Viper as the canonical
			// manager of config directives – it's too easy for things to get lost in (string) translation.
			bugIfError(
				cmd.Flags().Set(f.Name, viper.GetString(f.Name)),
				"could not set flag value")
		}
	})

	return nil
}

type APIClient interface {
	MyIP() (net.IP, error)
	MyIPWithContext(context.Context) (net.IP, error)
	UpdateAlias() (net.IP, error)
	UpdateAliasWithContext(context.Context) (net.IP, error)
}

var apiClient APIClient

func bootstrapAPIClient(cmd *cobra.Command) error {
	baseURL, err := cmd.Flags().GetString("api-url")
	bugIfError(err, "could not determine the API URL")

	apiKey, err := cmd.Flags().GetString("api-key")
	bugIfError(err, "could not determine the API key")

	apiClient = sdk.NewClient(baseURL, apiKey)
	return nil
}

// flagNameToEnvVar transforms a flag name to its matching environment variable name.
func flagNameToEnvVar(envVarPrefix, flagName string) string {
	envVarSuffix := strings.ReplaceAll(flagName, "-", "_")
	return fmt.Sprintf("%s_%s", strings.ToUpper(envVarPrefix), strings.ToUpper(envVarSuffix))
}

// bugIfError panics unless err is nil.
// Use this for unrecoverable failures due to (presumably) programmer error, i.e. "flag accessed but not defined"
func bugIfError(err error, msg string) {
	if err != nil {
		panic(fmt.Errorf("%s (this is a bug!) due to error: %w", msg, err))
	}
}
