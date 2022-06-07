package cli

import (
	"context"
	"fmt"
	"net"
	"path/filepath"

	"github.com/spf13/cobra"
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
	cmd.PersistentFlags().String(configFileSettingKey, "",
		"Explicitly set a config file (disables config file discovery)")
	cmd.PersistentFlags().String(configPathSettingKey, defaultConfigPath,
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

	// Bind all CLI flags to Viper
	_ = viper.BindPFlags(cmd.Flags())

	// Explicitly bind config-path and config-file env vars
	viper.BindEnv(configPathSettingKey, fmt.Sprintf("%s_CONFIG_PATH", envPrefix))
	viper.BindEnv(configFileSettingKey, fmt.Sprintf("%s_CONFIG_FILE", envPrefix))

	if viper.IsSet(configFileSettingKey) {
		configFilename := viper.GetString(configFileSettingKey)
		if !filepath.IsAbs(configFilename) {
			configFilename = filepath.Join(viper.GetString(configPathSettingKey), configFilename)
		}
		viper.SetConfigFile(configFilename)
	} else {
		viper.SetConfigName(defaultConfigFilename)
		viper.AddConfigPath(viper.GetString(configPathSettingKey))
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok || viper.IsSet(configFileSettingKey) {
			return err
		}
	}

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
	apiClient = sdk.NewClient(viper.GetString("api-url"), viper.GetString("api-key"))
	return nil
}
