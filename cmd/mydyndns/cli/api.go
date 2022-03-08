package cli

import (
	"github.com/spf13/cobra"
)

func newAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "mydyndns API client operations",
	}
}

func newAPIMyIPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "my-ip",
		Short: "Show the external-facing IP address",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return firstValidationError(cmd, validateAPIKey, validateBaseURL)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			myIP, err := apiClient.MyIP()
			if err != nil {
				return err
			}
			cmd.Println(myIP)
			return nil
		},
	}
}

func newAPIUpdateAliasCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-alias",
		Short: "Request a DNS update that points to the external-facing IP address",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return firstValidationError(cmd, validateAPIKey, validateBaseURL)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			myIP, err := apiClient.UpdateAlias()
			if err != nil {
				return err
			}
			cmd.Println(myIP)
			return nil
		},
	}
}
