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
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Name = cmd.Flags().Arg(0)
			return RunDeleteLayerCommand(f, options, out)
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
