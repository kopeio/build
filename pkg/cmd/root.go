package cmd

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"io"
	"path/filepath"
)

func Execute(out io.Writer) {
	goflag.CommandLine.Parse([]string{})

	dir := filepath.Join(os.Getenv("HOME"), ".imagebuilder")
	f := newFSFactory(dir)

	cmd := buildRootCommand(f, out)
	if err := cmd.Execute(); err != nil {
		ExitWithError(err)
	}
}

func ExitWithError(err error) {
	fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}

func buildRootCommand(f Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: "imagebuilder",
	}

	cmd.AddCommand(BuildCopyCommand(f, out))
	cmd.AddCommand(BuildCreateCommand(f, out))
	cmd.AddCommand(BuildDeleteCommand(f, out))
	cmd.AddCommand(BuildFetchCommand(f, out))
	cmd.AddCommand(BuildPushCommand(f, out))
	cmd.AddCommand(BuildSetCommand(f, out))

	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)

	return cmd
}