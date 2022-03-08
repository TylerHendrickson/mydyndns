package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func validatePollInterval(cmd *cobra.Command) error {
	if pollInterval, _ := cmd.Flags().GetDuration("interval"); pollInterval < minimumPollInterval {
		return fmt.Errorf("poll interval cannot be less than %s", minimumPollInterval)
	}
	return nil
}

func validateBaseURL(cmd *cobra.Command) error {
	if baseURL, _ := cmd.Flags().GetString("api-url"); baseURL == "" {
		return fmt.Errorf("missing API base URL directive")
	} else if !strings.HasPrefix(strings.ToLower(baseURL), "https://") {
		return fmt.Errorf("SSL is required for API Base URL (received %q)", baseURL)
	}
	return nil
}

func validateAPIKey(cmd *cobra.Command) error {
	if apiKey, _ := cmd.Flags().GetString("api-key"); apiKey == "" {
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
