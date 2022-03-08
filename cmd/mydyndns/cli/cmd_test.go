package cli

import (
	"bytes"
	"context"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	// Get rid of environment variables before running tests
	for _, env := range os.Environ() {
		if k := strings.Split(env, "=")[0]; strings.HasPrefix(k, envPrefix) {
			if err := os.Unsetenv(k); err != nil {
				panic(err)
			}
		}
	}

	os.Exit(m.Run())
}

func ExecuteContextC(ctx context.Context, cmd *cobra.Command, args ...string) (*cobra.Command, string, error) {
	return executorC(cmd, args, func() (*cobra.Command, error) { return cmd.ExecuteContextC(ctx) })
}

func ExecuteC(cmd *cobra.Command, args ...string) (*cobra.Command, string, error) {
	return executorC(cmd, args, cmd.ExecuteC)
}

func executorC(cmd *cobra.Command, args []string, fn func() (*cobra.Command, error)) (*cobra.Command, string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	c, err := fn()
	return c, buf.String(), err
}

type mockClient struct{ mock.Mock }

func (m *mockClient) MyIP() (ip net.IP, err error) { return m.coerceRV(m.Called()) }

func (m *mockClient) MyIPWithContext(context.Context) (ip net.IP, err error) {
	return m.coerceRV(m.Called())
}

func (m *mockClient) UpdateAlias() (ip net.IP, err error) {
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

func patchBootstrappedAPIClient(mocked APIClient, rootCmd *cobra.Command) {
	originalPersistentPreRunE := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		defer func() { apiClient = mocked }()
		return originalPersistentPreRunE(cmd, args)
	}
}
