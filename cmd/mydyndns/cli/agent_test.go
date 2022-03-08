package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func logLine2JSON(t *testing.T, lines []string, lineNo int) map[string]string {
	t.Helper()
	lineS := lines[lineNo]
	lineB := []byte(lineS)
	logData := map[string]string{}
	require.NoError(t, json.Unmarshal(lineB, &logData), "Error parsing JSON on line %d: %s", lineNo, lines[lineNo])
	return logData
}

func TestAgentStart(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		prepareContext         func() (context.Context, context.CancelFunc)
		expectedShutdownReason error
		prepareClient          func() *mockClient
		expectedCmdError       error
	}{
		{
			"shutdown after timeout",
			func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Millisecond)
			},
			context.DeadlineExceeded,
			func() *mockClient {
				client := new(mockClient)
				client.On("UpdateAliasWithContext").Return(net.ParseIP("1.2.3.4"), nil)
				return client
			},
			nil,
		},
		{
			"premature shutdown",
			func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			context.Canceled,
			func() *mockClient {
				client := new(mockClient)
				client.On("UpdateAliasWithContext").Return(nil, context.Canceled)
				return client
			},
			fmt.Errorf("failed to start agent: %w", context.Canceled),
		},
		{
			"eventual shutdown",
			func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				time.AfterFunc(time.Millisecond, cancel)
				return ctx, cancel
			},
			context.Canceled,
			func() *mockClient {
				client := new(mockClient)
				client.On("UpdateAliasWithContext").Return(net.ParseIP("2.3.4.5"), nil)
				return client
			},
			nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCLI()
			client := tt.prepareClient()
			patchBootstrappedAPIClient(client, cmd)

			ctx, cancel := tt.prepareContext()
			defer cancel()
			cmd.SetOut(new(bytes.Buffer))
			stdErr := new(bytes.Buffer)
			cmd.SetErr(stdErr)
			cmd.SetArgs([]string{
				"agent", "start",
				"--api-key=asdfjkl", "--api-url=https://example.com", "--log-json", "-vv",
			})
			cmd, err := cmd.ExecuteContextC(ctx)
			require.Equal(t, "start", cmd.Name())
			if tt.expectedCmdError == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.expectedCmdError.Error())
			}

			client.AssertExpectations(t)

			output := stdErr.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")
			if tt.expectedCmdError != nil {
				require.Equal(t, lines[len(lines)-1], fmt.Sprintf("Error: %s", tt.expectedCmdError))
				lines = lines[:len(lines)-1]
			}

			t.Run("log_startup", func(t *testing.T) {
				t.Run("line_0", func(t *testing.T) {
					record := logLine2JSON(t, lines, 0)
					assert.Equal(t, "Configured logger", record["msg"])
					assert.Equal(t, "debug", record["effective_level"])
				})
				t.Run("line_1", func(t *testing.T) {
					record := logLine2JSON(t, lines, 1)
					assert.Equal(t, "Initializing agent...", record["msg"])
				})
			})

			t.Run("log_shutdown", func(t *testing.T) {
				t.Run("line_-2", func(t *testing.T) {
					record := logLine2JSON(t, lines, len(lines)-2)
					if tt.expectedCmdError == nil {
						assert.Equal(t, "Shutdown requested", record["msg"])
					} else {
						assert.Equal(t, "Shutdown requested before start", record["msg"])
					}
				})
			})

			if tt.expectedCmdError == nil {
				t.Run("log_stopped", func(t *testing.T) {
					t.Run("line_-1", func(t *testing.T) {
						record := logLine2JSON(t, lines, len(lines)-1)
						assert.Equal(t, "Agent stopped", record["msg"], fmt.Sprintf("%s", record))
					})
				})
			}
		})
	}
}
