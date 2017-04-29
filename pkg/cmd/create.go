package cmd

import (
	"github.com/spf13/cobra"
	"io"
)

func BuildCreateCommand(f Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: "create",
	}

	cmd.AddCommand(BuildCreateLayerCommand(f, out))

	return cmd
}