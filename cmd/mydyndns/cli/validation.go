package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func validatePollInterval(cmd *cobra.Command) error {
	if pollInterval := viper.GetDuration("interval"); pollInterval < minimumPollInterval {
		return fmt.Errorf("poll interval cannot be less than %s", minimumPollInterval)
	}
	return nil
}

func validateBaseURL(cmd *cobra.Command) error {
	if baseURL := viper.GetString("api-url"); baseURL == "" {
		return fmt.Errorf("missing API base URL directive")
	} else if !strings.HasPrefix(strings.ToLower(baseURL), "https://") {
		return fmt.Errorf("SSL is required for API Base URL (received %q)", baseURL)
	}
	return nil
}

func validateAPIKey(cmd *cobra.Command) error {
	if apiKey := viper.GetString("api-key"); apiKey == "" {
		return fmt.Errorf("missing API key directive")
	}
	return nil
}

func firstValidationError(cmd *cobra.Command, validators ...func(*cobra.Command) error) error {
	for _, fn := range validators {
		if err := fn(cmd); err != nil {
			return err
		}
	}
	return nil
}
