package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

type DeleteLayerOptions struct {
	Name string
}

func BuildDeleteLayerCommand(f Factory, out io.Writer) *cobra.Command {
	options := &DeleteLayerOptions{}

	cmd := &cobra.Command{
		Use: "layer",
		Run: func(cmd *cobra.Command, args []string) {
			options.Name = cmd.Flags().Arg(0)
			if err := RunDeleteLayerCommand(f, options, out); err != nil {
				ExitWithError(err)
			}
		},
	}

	return cmd
}

func RunDeleteLayerCommand(f Factory, options *DeleteLayerOptions, out io.Writer) error {
	if options.Name == "" {
		return fmt.Errorf("layer name is required")
	}

	layerStore, err := f.LayerStore()
	if err != nil {
		return err
	}

	err = layerStore.DeleteLayer(options.Name)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Deleted layer %q\n", options.Name)
	return nil
}
