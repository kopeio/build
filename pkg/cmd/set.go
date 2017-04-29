package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"strings"
)

type SetOptions struct {
	Layer string
	Key   string
	Value []string
}

func BuildSetCommand(f Factory, out io.Writer) *cobra.Command {
	options := &SetOptions{}

	cmd := &cobra.Command{
		Use: "set",
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Layer = cmd.Flags().Arg(0)
			options.Key = cmd.Flags().Arg(1)
			options.Value = cmd.Flags().Args()
			if len(options.Value) >= 2 {
				options.Value = options.Value[2:]
			} else {
				options.Value = nil
			}
			return RunSetCommand(f, options, out)
		},
	}

	return cmd
}

func RunSetCommand(factory Factory, options *SetOptions, out io.Writer) error {
	if options.Layer == "" {
		return fmt.Errorf("layer is required")
	}
	if options.Key == "" {
		return fmt.Errorf("key is required")
	}
	if options.Value == nil {
		return fmt.Errorf("value is required")
	}

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

	switch strings.ToLower(options.Key) {
	case "workdir":
		if len(options.Value) != 1 {
			return fmt.Errorf("expected a single value for workdir")
		}
		meta.WorkingDir = options.Value[0]

	case "cmd":
		meta.Cmd = options.Value

	case "base":
		if len(options.Value) != 1 {
			return fmt.Errorf("expected a single value for base")
		}
		meta.Base = options.Value[0]

	default:
		return fmt.Errorf("unknown key %q", options.Key)
	}

	if err := l.SetOptions(meta); err != nil {
		return err
	}

	fmt.Fprintf(out, "Set %s=%s\n", options.Key, options.Value)
	return nil
}
