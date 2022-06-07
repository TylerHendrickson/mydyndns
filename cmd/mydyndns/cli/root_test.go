package cli

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapConfigConfigFileResolution(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := TempFile(t, tempDir, "*.toml")
	absFile := tempFile.Name()
	_, relFile := filepath.Split(tempFile.Name())
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
