package cmd

import (
	"github.com/spf13/cobra"
	"io"
	"fmt"
	"kope.io/imagebuilder/pkg/layers"
)

type CreateLayerOptions struct {
	Name string
	Base string
}

func BuildCreateLayerCommand(f Factory, out io.Writer) *cobra.Command {
	options := &CreateLayerOptions{}

	cmd := &cobra.Command{
		Use: "layer",
		RunE: func(cmd*cobra.Command, args []string) error {
			options.Name = cmd.Flags().Arg(0)
			return RunCreateLayerCommand(f, options, out)
		},
	}

	cmd.Flags().StringVar(&options.Base, "base", "", "specify base layer or image")


	return cmd
}

func RunCreateLayerCommand(f Factory, options *CreateLayerOptions, out io.Writer) error {
	if options.Name == "" {
		return fmt.Errorf("Name is required")
	}

	layerStore, err := f.LayerStore()
	if err != nil {
		return err
	}

	meta := layers.Options{}
	meta.Base = options.Base

	l, err := layerStore.CreateLayer(options.Name, meta)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Created layer %q\n", l.Name())
	return nil
}