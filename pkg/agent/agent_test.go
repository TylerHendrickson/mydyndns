package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockClient struct{ mock.Mock }

func (m *mockClient) MyIPWithContext(context.Context) (ip net.IP, err error) {
	return m.coerceRV(m.Called())
}

func (m *mockClient) UpdateAliasWithContext(context.Context) (ip net.IP, err error) {
	return m.coerceRV(m.Called())
}

func (m *mockClient) coerceRV(args mock.Arguments) (ip net.IP, err error) {
	if rvIP := args.Get(0); rvIP != nil {
		ip = rvIP.(net.IP)
	}
	if rvErr := args.Get(1); rvErr != nil {
		err = rvErr.(error)
	}
	return
}

func TestAgentRunWithFailedStartup(t *testing.T) {
	underlyingClientError := fmt.Errorf("alias update error")
	client := &mockClient{}
	client.On("UpdateAliasWithContext").Return(nil, underlyingClientError).Once()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Run(ctx, log.NewJSONLogger(io.Discard), client, time.Second)
	assert.ErrorIs(t, err, underlyingClientError)
	client.AssertExpectations(t)
}

func TestAgentRunWithPrematureShutdown(t *testing.T) {
	client := &mockClient{}
	client.On("UpdateAliasWithContext").Return(nil, fmt.Errorf("error: %w", context.Canceled)).Once()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	cancel()

	err := Run(ctx, log.NewJSONLogger(io.Discard), client, time.Second)
	assert.ErrorIs(t, err, context.Canceled)
	client.AssertNotCalled(t, "MyIPWithContext")
	client.AssertExpectations(t)
}

func TestAgentRun(t *testing.T) {
	client := &mockClient{}
	var expectedLogs []map[string]string
	expectedLogs = append(expectedLogs, map[string]string{"msg": "Initializing agent...", "level": "info"})

	for _, exp := range []struct{ patchMethod, rvIP, rvErr, logMsg string }{
		{patchMethod: "UpdateAliasWithContext", rvIP: "1.2.3.4", logMsg: "Initialized with IP address after DNS update"},
		{patchMethod: "MyIPWithContext", rvIP: "1.2.3.4", logMsg: "Fetched my IP address"},
		{patchMethod: "MyIPWithContext", rvIP: "9.8.7.6", logMsg: "Fetched my IP address"},
		{patchMethod: "UpdateAliasWithContext", rvIP: "9.8.7.6", logMsg: "Updated IP alias"},
		{patchMethod: "MyIPWithContext", rvIP: "9.8.7.6", logMsg: "Fetched my IP address"},
		{patchMethod: "MyIPWithContext", rvErr: "ip fetch error", logMsg: "Error fetching my IP address"},
		{patchMethod: "MyIPWithContext", rvIP: "2.3.4.5", logMsg: "Fetched my IP address"},
		{patchMethod: "UpdateAliasWithContext", rvErr: "alias update error", logMsg: "Error updating DNS alias"},
		{patchMethod: "MyIPWithContext", rvIP: "2.3.4.5", logMsg: "Fetched my IP address"},
	} {
		var (
			expectedLog = map[string]string{"msg": exp.logMsg, "level": "info"}
			rvIP        net.IP
			rvErr       error
		)
		if exp.rvIP != "" {
			rvIP = net.ParseIP(exp.rvIP)
			expectedLog["ip"] = exp.rvIP
		}
		if exp.rvErr != "" {
			rvErr = fmt.Errorf(exp.rvErr)
			expectedLog["error"] = exp.rvErr
			expectedLog["level"] = "error"
		}
		client.On(exp.patchMethod).Return(rvIP, rvErr).Once()
		expectedLogs = append(expectedLogs, expectedLog)
	}
	client.On("MyIPWithContext").Return(net.ParseIP("2.3.4.5"), nil)
	client.On("UpdateAliasWithContext").Return(net.ParseIP("2.3.4.5"), nil)

	logWriter := new(bytes.Buffer)
	logger := level.NewFilter(log.NewJSONLogger(logWriter), level.AllowInfo())
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err := Run(timeoutCtx, logger, client, 10*time.Millisecond)
	require.NoError(t, err)
	require.True(t, client.AssertExpectations(t))

	loggedOutput := logWriter.String()
	lines := strings.Split(strings.TrimSpace(loggedOutput), "\n")
	require.GreaterOrEqual(t, len(lines), len(expectedLogs),
		fmt.Sprintf("Not enough data was logged:\n%s", loggedOutput))

	for lineNo, expectedLogData := range expectedLogs {
		output := []byte(lines[lineNo])
		logData := map[string]string{}
		require.NoError(t, json.Unmarshal(output, &logData),
			"Error parsing JSON on line %d", lineNo)

		assert.Equal(t, expectedLogData["ip"], logData["ip"], "line %d", lineNo)
		assert.Equal(t, expectedLogData["error"], logData["error"], "line %d", lineNo)
		assert.Equal(t, expectedLogData["level"], logData["level"], "line %d", lineNo)
		assert.Equal(t, expectedLogData["msg"], logData["msg"], "line %d", lineNo)
		//fmt.Printf("%d: %s\n", lineNo, lines[lineNo])
	}
}
