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
