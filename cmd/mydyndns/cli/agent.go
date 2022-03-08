package cli

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/TylerHendrickson/mydyndns/internal"
	"github.com/TylerHendrickson/mydyndns/pkg/agent"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Controls the mydyndns agent",
	}
	return cmd
}

func newAgentStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Starts the agent (as a long-running process)",
		Long: strings.TrimSpace(`
starts a long-running agent process that periodically polls for the external-facing IP address of the host machine 
by querying a configured remote instance of the mydyndns API service. When a change in the external-facing IP address 
is detected, the remote service is notified so that associated DNS records are updated to point to the new IP.`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return firstValidationError(cmd, validateAPIKey, validateBaseURL, validatePollInterval)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logVerbosity, _ := cmd.Flags().GetCount("log-verbosity")
			logJSON, _ := cmd.Flags().GetBool("log-json")
			logger := internal.ConfigureLogger(logJSON, logVerbosity, cmd.ErrOrStderr())
			pollInterval, _ := cmd.Flags().GetDuration("interval")

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGHUP, syscall.SIGINT, os.Interrupt)
			defer stop()
			return agent.Run(ctx, logger, apiClient, pollInterval)
		},
	}
}
