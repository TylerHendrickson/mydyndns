// Package agent provides a long-running MyDynDNS agent for keeping DNS records up-to-date with a Client-reported
// apparent IP address. An Agent process is configured to check the apparent IP address at regular intervals.
// At each interval, the latest-retrieved IP address is compared to the previously-retrieved IP address.
// When the compared values differ, the Agent attempts to update DNS records accordingly.
package agent

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// The Client interface is satisfied by the client struct type from the MyDynDNS SDK.
type Client interface {
	UpdateAliasWithContext(ctx context.Context) (net.IP, error)
	MyIPWithContext(ctx context.Context) (net.IP, error)
}

// Run executes the agent until the provided context.Context is cancelled.
// When the agent fails to start, Run returns an error.
func Run(ctx context.Context, logger log.Logger, client Client, pollInterval time.Duration) error {
	// Ensure the logger is safe for concurrent use
	logger = log.NewSyncLogger(logger)

	// Perform an initial blind update and provide the detected IP as the starting point to monitor against
	level.Info(logger).Log("msg", "Initializing agent...")
	startIP, err := client.UpdateAliasWithContext(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			level.Warn(logger).Log("msg", "Shutdown requested before start", "reason", ctxErr)
		}
		level.Error(logger).Log("msg", "Error getting initial IP address", "error", err)
		return fmt.Errorf("failed to start agent: %w", err)
	}
	level.Info(logger).Log("msg", "Initialized with IP address after DNS update", "ip", startIP.String())

	wg := sync.WaitGroup{}
	ips := make(chan net.IP, 1)

	// Enter the long-running agent refresh loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		pollIP(ctx, log.With(logger, "agent_operation", "refresh"), client, pollInterval, ips)
	}()

	// Enter the long-running agent update loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		updateDNS(ctx, log.With(logger, "agent_operation", "update"), client, startIP, ips)
	}()

	// Wait for agent goroutines to finish
	wg.Wait()
	level.Warn(logger).Log("msg", "Agent stopped")
	return nil
}

// pollIP retrieves the apparent Client-reported IP address at regular intervals and sends the retrieved values
// to the given channel.
// Poll operations continue indefinitely until the provided Context is done.
func pollIP(ctx context.Context, logger log.Logger, client Client, interval time.Duration, polledIPs chan<- net.IP) {
	level.Debug(logger).Log("msg", "Starting periodic refresh", "interval", interval)
	ticker := time.NewTicker(interval)
	for {
		select {
		case tick := <-ticker.C:
			tickLogger := log.With(logger, "trigger_ts", tick.Format(time.RFC3339Nano))
			level.Debug(tickLogger).Log("msg", "Fetching my IP address...")
			myIP, err := client.MyIPWithContext(ctx)
			if err != nil {
				level.Error(tickLogger).Log("msg", "Error fetching my IP address", "error", err)
			} else {
				level.Info(tickLogger).Log("msg", "Fetched my IP address", "ip", myIP.String())
				polledIPs <- myIP
			}

		case <-ctx.Done():
			level.Debug(logger).Log("msg", "Shutdown requested", "reason", ctx.Err())
			ticker.Stop()
			return
		}
	}
}

// updateDNS monitors the given channel for new IP address values, and requests the Client to update DNS records
// whenever the newly-received IP address differs from the previously-received value.
// The first value is determined by the given startIP.
// This function will indefinitely wait for new IP addresses until the provided Context is done.
func updateDNS(ctx context.Context, logger log.Logger, client Client, startIP net.IP, latestIPs <-chan net.IP) {
	previousIP := startIP

	level.Debug(logger).Log("msg", "Waiting for refreshed IP address", "starting_ip", startIP)
	for {
		select {
		case latestIP := <-latestIPs:
			if !latestIP.Equal(previousIP) {
				level.Debug(logger).Log("msg", "IP address change detected",
					"previous", previousIP.String(), "new", latestIP.String())
				if aliasIP, err := client.UpdateAliasWithContext(ctx); err != nil {
					level.Error(logger).Log("msg", "Error updating DNS alias", "error", err)
				} else {
					level.Info(logger).Log("msg", "Updated IP alias", "ip", aliasIP.String())
					previousIP = aliasIP
				}
			} else {
				level.Debug(logger).Log("msg", "No change in latest IP address", "ip", latestIP)
			}

		case <-ctx.Done():
			level.Debug(logger).Log("msg", "Shutdown requested", "reason", ctx.Err())
			return
		}
	}
}
