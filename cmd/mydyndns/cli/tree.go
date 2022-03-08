package cli

import (
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"

	"github.com/TylerHendrickson/mydyndns/internal"
)

func newCommandTreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "command-tree",
		Hidden: true,
		Long: `Prints an ASCII tree representation of the nested (sub)command hierarchy. 
Note that output excludes this command, "help", "completion", and deprecated/hidden commands.`,
		Run: func(cmd *cobra.Command, args []string) {
			exclusions := internal.NewStringCollection("completion")
			tree := cmdToTree(cmd.Root(), func(c *cobra.Command) bool {
				return !exclusions.Contains(c.Name()) && c.IsAvailableCommand()
			})
			cmd.Print(tree.String())
		},
	}

	return cmd
}

func cmdToTree(cmd *cobra.Command, f func(*cobra.Command) bool) treeprint.Tree {
	var buildTree func(treeprint.Tree, *cobra.Command)
	buildTree = func(t treeprint.Tree, c *cobra.Command) {
		for _, child := range c.Commands() {
			if f(child) {
				buildTree(t.AddBranch(child.Name()), child)
			}
		}
	}

	tree := treeprint.NewWithRoot(cmd.Name())
	buildTree(tree, cmd)
	return tree
}
