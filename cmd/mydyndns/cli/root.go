package cli

import (
	"context"
	"fmt"
	"net"
	"os"
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
	requireConfigFile, err := bootstrapConfigFile(cmd, viper.GetViper())
	bugIfError(err, "could not bootstrap config file")

	if err = func() (e error) {
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
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok || requireConfigFile {
				return err
			}
		}
		return nil
	}(); err != nil {
		return err
	}

	// Matching environment variables must have prefix MYDYNDNS_
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --foo-bar to MYDYNDNS_FOO_BAR
		_ = viper.BindEnv(f.Name, flagNameToEnvVar(envPrefix, f.Name))

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && viper.IsSet(f.Name) {
			bugIfError(
				cmd.Flags().Set(f.Name, viper.GetString(f.Name)),
				"could not set flag value")
		}
	})

	return nil
}

// bootstrapConfigFile inspects the *cobra.Command flags and environment variables
// and instructs the *viper.Viper to read configuration from a file based on these settings.
// The first return value is a boolean which indicates whether a config file is explicitly
// set (either via flag or environment variables).
// The second return value is an error that, if non-nil, indicates that the flag could not be determined
// (due to a typo/bug).
func bootstrapConfigFile(cmd *cobra.Command, v *viper.Viper) (bool, error) {
	const (
		configPathFlagName = "config-path"
		configFileFlagName = "config-file"
	)

	configSearchPath, err := cmd.Flags().GetString(configPathFlagName)
	if err != nil {
		return false, err
	}
	if !cmd.Flag(configPathFlagName).Changed {
		if envConfigSearchPath, isSet := os.LookupEnv(fmt.Sprintf("%s_CONFIG_PATH", envPrefix)); isSet {
			configSearchPath = envConfigSearchPath
		}
	}

	var explicitConfigFile = false
	configFilename, err := cmd.Flags().GetString(configFileFlagName)
	if err != nil {
		return false, err
	}
	if cmd.Flag(configFileFlagName).Changed {
		explicitConfigFile = true
	} else if envConfigFile, isSet := os.LookupEnv(fmt.Sprintf("%s_CONFIG_FILE", envPrefix)); isSet {
		explicitConfigFile = true
		configFilename = envConfigFile
	}

	if explicitConfigFile {
		if !filepath.IsAbs(configFilename) {
			configFilename = filepath.Join(configSearchPath, configFilename)
		}
		v.SetConfigFile(configFilename)
	} else {
		v.SetConfigName(defaultConfigFilename)
		v.AddConfigPath(configSearchPath)
	}

	return explicitConfigFile, nil
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
