package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

type EnvOptions struct {
	Layer string
	Key   string
	Value string
}

func BuildEnvCommand(f Factory, out io.Writer) *cobra.Command {
	options := &EnvOptions{}

	cmd := &cobra.Command{
		Use: "env",
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().NArg() != 3 {
				ExitWithError(fmt.Errorf("syntax: <layer> <key> <value>"))
				return
			}

			options.Layer = cmd.Flags().Arg(0)
			options.Key = cmd.Flags().Arg(1)
			options.Value = cmd.Flags().Arg(2)
			if err := RunEnvCommand(f, options, out); err != nil {
				ExitWithError(err)
			}
		},
	}

	return cmd
}

func RunEnvCommand(factory Factory, options *EnvOptions, out io.Writer) error {
	if options.Layer == "" {
		return fmt.Errorf("layer is required")
	}
	if options.Key == "" {
		return fmt.Errorf("key is required")
	}
	// Value is _not_ required (I think!)

	layerStore, err := factory.LayerStore()
	if err != nil {
		return err
	}

	l, err := layerStore.FindLayer(options.Layer)
	if err != nil {
		return err
	}

	if l == nil {
		return fmt.Errorf("layer %q does not exist", options.Layer)
	}

	meta, err := l.GetOptions()
	if err != nil {
		return err
	}

	if meta.Env == nil {
		meta.Env = make(map[string]string)
	}
	meta.Env[options.Key] = options.Value

	if err := l.SetOptions(meta); err != nil {
		return err
	}

	fmt.Fprintf(out, "ENV %s=%s\n", options.Key, options.Value)
	return nil
}
