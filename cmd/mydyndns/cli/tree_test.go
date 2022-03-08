package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommandTreeCmd(t *testing.T) {
	cmd, out, err := ExecuteC(newCLI(), "command-tree")
	require.Equal(t, "command-tree", cmd.Name())
	require.NoError(t, err)
	require.True(t, cmd.Hidden, "Expected \"command-tree\" command to be hidden in CLI")

	assert.True(t, strings.HasPrefix(out, "mydyndns"),
		"Tree root node should be named after root command")
	assert.NotContains(t, out, "command-tree",
		"Tree output should exclude hidden commands")
	assert.NotContains(t, out, "completion",
		"Tree output should exclude built-in \"completion\" command")
	assert.NotContains(t, out, "help",
		"Tree output should exclude build-in \"help\" command")
}

func TestCmdToTree(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	a := &cobra.Command{Use: "a"}
	root.AddCommand(a)
	aa := &cobra.Command{Use: "aa"}
	ab := &cobra.Command{Use: "ab"}
	abc := &cobra.Command{Use: "abc"}
	ab.AddCommand(abc)
	ac := &cobra.Command{Use: "ac"}
	aca := &cobra.Command{Use: "aca"}
	ac.AddCommand(aca)
	acaa := &cobra.Command{Use: "acaa"}
	acab := &cobra.Command{Use: "acab"}
	aca.AddCommand(acaa, acab)
	a.AddCommand(aa, ab, ac)
	b := &cobra.Command{Use: "b"}
	root.AddCommand(b)
	ba := &cobra.Command{Use: "ba"}
	b.AddCommand(ba)
	c := &cobra.Command{Use: "c"}
	root.AddCommand(c)

	for _, tt := range []struct {
		name         string
		expectedTree string
		filter       func(*cobra.Command) bool
	}{
		{
			"include all",
			`root
├── a
│   ├── aa
│   ├── ab
│   │   └── abc
│   └── ac
│       └── aca
│           ├── acaa
│           └── acab
├── b
│   └── ba
└── c
`,
			func(command *cobra.Command) bool { return true },
		},
		{
			"exclude ab",
			`root
├── a
│   ├── aa
│   └── ac
│       └── aca
│           ├── acaa
│           └── acab
├── b
│   └── ba
└── c
`,
			func(command *cobra.Command) bool { return !strings.HasPrefix(command.Name(), "ab") },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tree := cmdToTree(root, tt.filter)
			assert.Equal(t, tt.expectedTree, tree.String())
		})
	}
}
