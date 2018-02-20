package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func BuildCreateCommand(f Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: "create",
	}

	cmd.AddCommand(BuildCreateLayerCommand(f, out))

	return cmd
}
