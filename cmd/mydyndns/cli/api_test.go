package cli

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiSubcommands(t *testing.T) {
	// NB: The my-ip and update-alias subcommands behave the same,
	// but they call different underlying client methods
	for _, subcommand := range []string{"my-ip", "update-alias"} {
		t.Run(subcommand, func(t *testing.T) {
			for _, tt := range []struct {
				name          string
				flags         []string
				ip            net.IP
				validationErr error
				clientErr     error
			}{
				{
					name:  "IP on client operation success",
					flags: []string{"--api-url=https://example.com", "--api-key=asdfjkl"},
					ip:    net.ParseIP("1.2.3.4"),
				},
				{
					name:      "error on client operation failure",
					flags:     []string{"--api-url=https://bad+hostname!", "--api-key=asdfjkl"},
					clientErr: url.InvalidHostError("bad+hostname!"),
				},
				{
					name:          "error on missing API URL",
					flags:         []string{"--api-key=asdfjkl"},
					validationErr: fmt.Errorf("missing API base URL directive"),
				},
				{
					name:  "error on non-SSL API URL",
					flags: []string{"--api-url=http://example.com", "--api-key=asdfjkl"},
					validationErr: fmt.Errorf("SSL is required for API Base URL (received %q)",
						"http://example.com"),
				},
				{
					name:          "error on missing API key",
					flags:         []string{"--api-url=https://example.com"},
					validationErr: fmt.Errorf("missing API key directive"),
				},
			} {
				cmd := newCLI()
				client := new(mockClient)
				patchBootstrappedAPIClient(client, cmd)
				switch subcommand {
				case "my-ip":
					client.On("MyIP").Return(tt.ip, tt.clientErr).Once()
				case "update-alias":
					client.On("UpdateAlias").Return(tt.ip, tt.clientErr).Once()
				default:
					require.FailNow(t, "unknown subcommand")
				}

				args := append([]string{"api", subcommand}, tt.flags...)
				cmd, out, err := ExecuteC(cmd, args...)
				require.Equal(t, subcommand, cmd.Name())

				t.Run(tt.name, func(t *testing.T) {
					if tt.validationErr != nil {
						assert.EqualError(t, err, tt.validationErr.Error())
					} else {
						if tt.clientErr != nil {
							assert.EqualError(t, err, tt.clientErr.Error())
						} else {
							assert.NoError(t, err)
							assert.Equal(t, tt.ip.String(), strings.TrimSpace(out))
						}
						client.AssertExpectations(t)
					}
				})
			}
		})
	}
}
