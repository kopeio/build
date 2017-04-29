package cmd

import (
	"github.com/spf13/cobra"
	"io"
)

func BuildDeleteCommand(f Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: "delete",
	}

	cmd.AddCommand(BuildDeleteLayerCommand(f, out))

	return cmd
}
