package cli

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigWriteCmd(t *testing.T) {
	type TT struct {
		name                    string
		configDir               string
		destinationArgs         []string
		otherArgs               []string
		quiet                   bool
		expectedOutputFilenames []string
		expectedConfig          map[string]interface{}
		expectedError           func(tt TT) error
	}
	returnsNil := func(_ TT) error { return nil }

	for _, tt := range []TT{
		{
			"defaults",
			TempDir(t),
			[]string{"toml"},
			[]string{"--defaults"},
			false,
			[]string{"mydyndns.toml"},
			map[string]interface{}{
				"api-key":       "",
				"api-url":       "",
				"interval":      defaultPollInterval.String(),
				"log-json":      "false",
				"log-verbosity": "0",
			},
			returnsNil,
		},
		{
			"effective non-default config",
			TempDir(t),
			[]string{"toml"},
			[]string{
				"--api-key=asdfjkl",
				"--api-url=https://example.com",
				"--interval=24h",
				"--log-json",
				"--log-verbosity=2",
				"--validate",
			},
			false,
			[]string{"mydyndns.toml"},
			map[string]interface{}{
				"api-key":       "asdfjkl",
				"api-url":       "https://example.com",
				"interval":      (time.Hour * 24).String(),
				"log-json":      "true",
				"log-verbosity": "2",
			},
			returnsNil,
		},
		{
			"defaults with nonstandard filename",
			TempDir(t),
			[]string{"foobar.yaml"},
			[]string{"--defaults"},
			false,
			[]string{"foobar.yaml"},
			map[string]interface{}{
				"api-key":       "",
				"api-url":       "",
				"interval":      defaultPollInterval.String(),
				"log-json":      "false",
				"log-verbosity": "0",
			},
			returnsNil,
		},
		{
			"multiple files with defaults",
			TempDir(t),
			[]string{"toml", "foobar.yaml", "json", "yml"},
			[]string{"--defaults"},
			false,
			[]string{"mydyndns.toml", "foobar.yaml", "mydyndns.json", "mydyndns.yml"},
			map[string]interface{}{
				"api-key":       "",
				"api-url":       "",
				"interval":      defaultPollInterval.String(),
				"log-json":      "false",
				"log-verbosity": "0",
			},
			returnsNil,
		},
		{
			"safe write fails",
			TempDir(t),
			[]string{"foobar.yaml", "foobar.yaml"},
			[]string{"--defaults", "--safe"},
			false,
			[]string{"foobar.yaml"},
			map[string]interface{}{
				"api-key":       "",
				"api-url":       "",
				"interval":      defaultPollInterval.String(),
				"log-json":      "false",
				"log-verbosity": "0",
			},
			func(tt TT) error {
				return viper.ConfigFileAlreadyExistsError(filepath.Join(tt.configDir, "foobar.yaml"))
			},
		},
		{
			"fail when validation is requested",
			TempDir(t),
			[]string{"toml"},
			[]string{
				"--api-url=https://example.com",
				"--interval=24h",
				"--log-json",
				"--log-verbosity=2",
				"--validate",
			},
			false,
			nil,
			nil,
			func(tt TT) error {
				return fmt.Errorf("missing API key directive")
			},
		},
		{
			"fail when config type is unsupported",
			TempDir(t),
			[]string{"notarealconfigtype"},
			[]string{"--defaults"},
			false,
			nil,
			nil,
			func(tt TT) error {
				return viper.UnsupportedConfigError("notarealconfigtype")
			},
		},
		{
			"requires at least 1 argument",
			TempDir(t),
			nil,
			[]string{"--defaults"},
			false,
			nil,
			nil,
			func(tt TT) error {
				return cobra.MinimumNArgs(1)(nil, []string{})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"config", "write"}, tt.destinationArgs...)
			args = append(args, tt.otherArgs...)
			args = append(args, fmt.Sprintf("--directory=%s", tt.configDir))

			cmd, out, err := ExecuteC(newCLI(), args...)
			require.Equal(t, "write", cmd.Name())

			if expectedErr := tt.expectedError(tt); err != nil {
				assert.EqualError(t, err, expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			lines := strings.Split(strings.TrimSpace(out), "\n")
			if !tt.quiet && tt.expectedError == nil {
				require.Equal(t, len(tt.expectedOutputFilenames), len(lines))
			}

			for i := range tt.expectedOutputFilenames {
				expectedOutputFilename := filepath.Join(tt.configDir, tt.expectedOutputFilenames[i])
				t.Run(tt.expectedOutputFilenames[i], func(t *testing.T) {
					if !tt.quiet {
						require.Equal(t, expectedOutputFilename, lines[i])
					}
					v := viper.New()
					v.SetConfigFile(expectedOutputFilename)
					require.NoError(t, v.ReadInConfig())
					assert.Equal(t, tt.expectedConfig, v.AllSettings())
				})
			}
		})
	}
}

func TestConfigWriteCmdArgCompletion(t *testing.T) {
	for _, tt := range []struct {
		name                string
		cmd                 *cobra.Command
		inputArgs           []string
		toComplete          string
		expectedCompletions []string
		directive           cobra.ShellCompDirective
	}{
		{
			"no repeat suggestions",
			newConfigWriteCmd(),
			[]string{"toml"},
			"tom",
			func() (comps []string) {
				for _, ext := range viper.SupportedExts {
					if ext != "toml" {
						comps = append(comps, ext)
					}
				}
				return
			}(),
			cobra.ShellCompDirectiveDefault,
		},
		{
			"completes extension",
			newConfigWriteCmd(),
			nil,
			"foobar.tom",
			append(viper.SupportedExts, "foobar.toml"),
			cobra.ShellCompDirectiveDefault,
		},
		{
			"safe mode does not complete existing filenames",
			func() *cobra.Command {
				cmd := newConfigWriteCmd()
				cmd.Flags().Set("safe", "true")
				return cmd
			}(),
			nil,
			"yam",
			viper.SupportedExts,
			cobra.ShellCompDirectiveNoFileComp,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, d := tt.cmd.ValidArgsFunction(tt.cmd, tt.inputArgs, tt.toComplete)
			assert.ElementsMatch(t, tt.expectedCompletions, c)
			assert.Equal(t, tt.directive, d, "Unexpected shell comp directive returned")
		})
	}
}

func TestConfigTypesListCmd(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cmd, out, err := ExecuteC(newCLI(), "config", "types", "list")

		require.Equal(t, "list", cmd.Name())
		require.Nil(t, err)
		outList := strings.Split(out[strings.Index(out, ":")+1:], ", ")
		for i, it := range outList {
			outList[i] = strings.TrimSpace(it)
		}
		assert.ElementsMatch(t, outList, viper.SupportedExts)
	})

	t.Run("bare", func(t *testing.T) {
		cmd, out, err := ExecuteC(newCLI(), "config", "types", "list", "--bare")

		require.Equal(t, "list", cmd.Name())
		require.Nil(t, err)
		outList := strings.Split(strings.TrimSpace(out), "\n")
		assert.ElementsMatch(t, outList, viper.SupportedExts)
	})
}

func TestConfigTypesCheckCmd(t *testing.T) {
	for _, tt := range []struct {
		check string
		err   error
	}{
		{"bespokeformat", viper.UnsupportedConfigError("bespokeformat")},
		{"mydyndns.bespokeformat", viper.UnsupportedConfigError("bespokeformat")},
		{"json", nil}, {"mydyndns.json", nil},
		{"toml", nil}, {"mydyndns.toml", nil},
		{"yaml", nil}, {"mydyndns.yaml", nil},
		{"yml", nil}, {"mydyndns.yml", nil},
		{"properties", nil}, {"mydyndns.properties", nil},
		{"props", nil}, {"mydyndns.props", nil},
		{"prop", nil}, {"mydyndns.prop", nil},
		{"dotenv", nil}, {"mydyndns.dotenv", nil}, {".dotenv", nil},
		{"env", nil}, {"mydyndns.env", nil}, {".env", nil},
		{"ini", nil}, {"mydyndns.ini", nil},
	} {
		t.Run(tt.check, func(t *testing.T) {
			cmd, _, err := ExecuteC(newCLI(), "config", "types", "check", tt.check)
			assert.Equal(t, "check", cmd.Name())
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestConfigShowCmd(t *testing.T) {
	makeExpectedConfig := func(apiURL, apiKey, configFile, configPath, interval, logJson, logVerbosity string) map[string]string {
		return map[string]string{
			"api-url":       fmt.Sprintf("%q", apiURL),
			"api-key":       fmt.Sprintf("%q", apiKey),
			"config-file":   fmt.Sprintf("%q", configFile),
			"config-path":   fmt.Sprintf("%q", configPath),
			"interval":      fmt.Sprintf("%q", interval),
			"log-json":      fmt.Sprintf("%q", logJson),
			"log-verbosity": fmt.Sprintf("%q", logVerbosity),
		}
	}

	configDir := TempDir(t)
	configFile, err := ioutil.TempFile(configDir, "*.toml")
	require.NoError(t, err)

	for _, tt := range []struct {
		name     string
		execute  func(t *testing.T, cmd *cobra.Command, args ...string) (*cobra.Command, string, error)
		expected map[string]string
	}{
		{
			"flags",
			func(t *testing.T, cmd *cobra.Command, args ...string) (*cobra.Command, string, error) {
				args = append(args,
					"--api-url=https://example.com/Test-flags",
					"--api-key=my-api-key",
					"--interval=2m",
					"--log-json=true",
					"--log-verbosity=1",
				)
				return ExecuteC(cmd, args...)
			},
			makeExpectedConfig(
				"https://example.com/Test-flags",
				"my-api-key",
				"",
				".",
				fmt.Sprint(time.Minute*2),
				"true",
				"1",
			),
		},
		{
			"defaults",
			func(t *testing.T, cmd *cobra.Command, args ...string) (*cobra.Command, string, error) {
				return ExecuteC(cmd, args...)
			},
			makeExpectedConfig("", "", "", ".", fmt.Sprint(defaultPollInterval), "false", "0"),
		},
		{
			"file",
			func(t *testing.T, cmd *cobra.Command, args ...string) (*cobra.Command, string, error) {
				v := viper.New()
				v.Set("api-url", "https://example.com/Test-file")
				v.Set("api-key", "some-api-key")
				v.Set("interval", time.Hour*12)
				v.Set("log-json", true)
				v.Set("log-verbosity", 2)
				require.NoError(t, v.WriteConfigAs(configFile.Name()))
				args = append(args, fmt.Sprintf("--config-file=%s", configFile.Name()), fmt.Sprintf("--config-path=%s", configDir))
				return ExecuteC(cmd, args...)
			},
			makeExpectedConfig(
				"https://example.com/Test-file",
				"some-api-key",
				configFile.Name(),
				configDir,
				fmt.Sprint(time.Hour*12),
				"true",
				"2",
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd, out, err := tt.execute(t, newCLI(), "config", "show")
			t.Cleanup(func() { viper.Reset() })
			require.Equal(t, "show", cmd.Name())
			require.NoError(t, err)

			for num, line := range strings.Split(strings.TrimSpace(out), "\n") {
				dV := strings.SplitN(line, " = ", 2)
				require.Len(t, dV, 2,
					"Could not parse effective config in CLI output on line %d", num)
				directive, value := dV[0], dV[1]
				t.Run(directive, func(t *testing.T) {
					assert.Equal(t, tt.expected[directive], value,
						"config directive %q has unexpected value", directive)
				})
			}
		})
	}
}

func TestConfigValidateCmd(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		err  error
	}{
		{
			"Missing API Key",
			[]string{
				"--api-url=https://example.com",
				"--interval=1h",
			},
			fmt.Errorf("missing API key directive"),
		},
		{
			"Missing API base URL",
			[]string{
				"--api-key=asdfjkl",
				"--interval=1h",
			},
			fmt.Errorf("missing API base URL directive"),
		},
		{
			"Non-SSL API base URL",
			[]string{
				"--api-key=asdfjkl",
				"--api-url=http://example.com",
				"--interval=1h",
			},
			fmt.Errorf("SSL is required for API Base URL (received %q)", "http://example.com"),
		},
		{
			"Poll interval below min threshold",
			[]string{
				"--api-key=asdfjkl",
				"--api-url=https://example.com",
				"--interval=1ms",
			},
			fmt.Errorf("poll interval cannot be less than %s", minimumPollInterval),
		},
		{
			"Valid configuration",
			[]string{
				"--api-key=asdfjkl",
				"--api-url=https://example.com",
				fmt.Sprintf("--interval=%s", minimumPollInterval),
			},
			nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"config", "validate"}, tt.args...)
			cmd, output, err := ExecuteC(newCLI(), args...)
			require.Equal(t, "validate", cmd.Name())
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "", output)
			}
		})
	}
}
