package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapConfigConfigFileResolution(t *testing.T) {
	tempDir := t.TempDir()
	tempFile, err := ioutil.TempFile(tempDir, "*.toml")
	absFile := tempFile.Name()
	_, relFile := filepath.Split(tempFile.Name())
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(tempFile.Name()))

	for _, tt := range []struct {
		name, configPath, configFile, expectedConfigFile string
		expectedError                                    error
	}{
		{
			"config-path ignored when config-file is absolute",
			"/foo/bar",
			absFile,
			absFile,
			nil,
		},
		{
			"config-path used when config-file is relative",
			tempDir,
			relFile,
			relFile,
			nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := ExecuteC(newCLI(), "config", "show",
				fmt.Sprintf("--config-file=%s", tt.configFile),
				fmt.Sprintf("--config-path=%s", tt.configPath))
			require.Equal(t, cmd.Name(), "show")
			require.NoError(t, err)

			configFile, err := cmd.Flags().GetString("config-file")
			require.NoError(t, err)
			assert.Equal(t, tt.expectedConfigFile, configFile)
		})
	}
}

func TestBootstrapConfigParsePanicFailsGracefully(t *testing.T) {
	tempDir := t.TempDir()
	tempFile, err := ioutil.TempFile(tempDir, "*.toml")
	require.NoError(t, err)
	_, err = tempFile.Write([]byte("a=1\nb=\nc=3"))
	require.NoError(t, err)

	v := viper.New()
	v.SetConfigFile(tempFile.Name())
	var panicMsg string
	func() {
		defer func() {
			r := recover()
			require.NotNil(t, r,
				"no panic reading corrupt toml file – perhaps this test is no longer needed?")
			panicMsg = fmt.Sprintf("%s", r)
		}()
		v.ReadInConfig()
	}()

	cmd, _, err := ExecuteC(newCLI(), "config", "show", fmt.Sprintf("--config-file=%s", tempFile.Name()))
	require.Equal(t, cmd.Name(), "show")
	assert.EqualError(t, err, fmt.Sprintf(
		"unrecoverable error reading (possibly corrupt) config file %q due to underlying error: %q",
		tempFile.Name(), panicMsg))
}

func TestBootstrapConfigFile(t *testing.T) {
	type expectedResult struct {
		isSet bool
		err   error
	}

	restoreEnv := func(t *testing.T, k, v string) {
		t.Helper()
		restoreValue, doRestore := os.LookupEnv(k)
		if doRestore {
			t.Cleanup(func() {
				if err := os.Setenv(k, restoreValue); err != nil {
					t.Fatal(err)
				}
			})
		} else {
			t.Cleanup(func() {
				if err := os.Unsetenv(k); err != nil {
					t.Fatal(err)
				}
			})
		}
		require.NoError(t, os.Setenv(k, v))
	}

	for _, tt := range []struct {
		name     string
		setup    func(t *testing.T, cmd *cobra.Command)
		expected expectedResult
	}{
		{
			"undefined config-file flag",
			func(t *testing.T, cmd *cobra.Command) {
				cmd.Flags().String("config-path", defaultConfigPath, "config search path")
			},
			expectedResult{false, fmt.Errorf("flag accessed but not defined: config-file")},
		},
		{
			"undefined config-path flag",
			func(t *testing.T, cmd *cobra.Command) {
				cmd.Flags().String("config-file", "my-config.toml", "config file")
			},
			expectedResult{false, fmt.Errorf("flag accessed but not defined: config-path")},
		},
		{
			"flag provides implicit config file",
			func(t *testing.T, cmd *cobra.Command) {
				cmd.Flags().String("config-path", defaultConfigPath, "config search path")
				cmd.Flags().String("config-file", "my-config.toml", "config file")
			},
			expectedResult{false, nil},
		},
		{
			"flag provides explicit config file",
			func(t *testing.T, cmd *cobra.Command) {
				cmd.Flags().String("config-path", defaultConfigPath, "config search path")
				cmd.Flags().String("config-file", "my-config.toml", "config file")
				if err := cmd.Flags().Parse([]string{"--config-file=my-config-file.toml"}); err != nil {
					t.Fatal(err)
				}
			},
			expectedResult{true, nil},
		},
		{
			"env provides explicit config search path",
			func(t *testing.T, cmd *cobra.Command) {
				cmd.Flags().String("config-path", defaultConfigPath, "config search path")
				cmd.Flags().String("config-file", "my-config.toml", "config file")
				restoreEnv(t, "MYDYNDNS_CONFIG_PATH", "/config")
			},
			expectedResult{false, nil},
		},
		{
			"env provides explicit config file",
			func(t *testing.T, cmd *cobra.Command) {
				cmd.Flags().String("config-path", defaultConfigPath, "config search path")
				cmd.Flags().String("config-file", "my-config.toml", "config file")
				restoreEnv(t, "MYDYNDNS_CONFIG_FILE", "my-config-file.toml")
			},
			expectedResult{true, nil},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			tt.setup(t, cmd)

			isSet, err := bootstrapConfigFile(cmd, viper.New())

			assert.Equal(t, tt.expected.isSet, isSet)
			if expectedErr := tt.expected.err; expectedErr != nil {
				assert.EqualError(t, err, expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFlagNameToEnvVar(t *testing.T) {
	for i, tt := range []struct {
		prefix, flagName, expected string
	}{
		{"foo", "bar", "FOO_BAR"},
		{"FOO", "bar-baz", "FOO_BAR_BAZ"},
		{"foo", "bar-BAZ-buzz", "FOO_BAR_BAZ_BUZZ"},
		{"foo_bar", "BAZ_BUZZ", "FOO_BAR_BAZ_BUZZ"},
		{"foo", "bar-baz_buzz", "FOO_BAR_BAZ_BUZZ"},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			actual := flagNameToEnvVar(tt.prefix, tt.flagName)

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestBugIfErrorHelper(t *testing.T) {
	t.Run("panics when err is present", func(t *testing.T) {
		assert.PanicsWithError(t, "could not do the thing (this is a bug!) due to error: oh no", func() {
			bugIfError(fmt.Errorf("oh no"), "could not do the thing")
		})
	})

	t.Run("panics when undefined flag is accessed", func(t *testing.T) {
		cmd := newCLI()
		flagName := "surely-this-flag-does-not-exist"
		require.Nil(t, cmd.Flags().Lookup(flagName), "Somehow this flag exists")

		_, err := cmd.Flags().GetString("surely-this-flag-does-not-exist")
		assert.PanicsWithError(t, fmt.Sprintf("could not access a flag (this is a bug!) due to error: %s", err),
			func() { bugIfError(err, "could not access a flag") },
		)
	})

	t.Run("does nothing when err is nil", func(t *testing.T) {
		assert.NotPanics(t, func() {
			bugIfError(nil, "could not do the thing")
		})
	})
}
