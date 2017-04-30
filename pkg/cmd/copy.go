package cmd

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"kope.io/imagebuilder/pkg/layers"
	"os"
	"path/filepath"
	"strings"
)

type CopyOptions struct {
	Source string
	Dest   string
}

func BuildCopyCommand(f Factory, out io.Writer) *cobra.Command {
	options := &CopyOptions{}

	cmd := &cobra.Command{
		Use: "cp",
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Source = cmd.Flags().Arg(0)
			options.Dest = cmd.Flags().Arg(1)
			return RunCopyCommand(f, options, out)
		},
	}

	return cmd
}

func RunCopyCommand(factory Factory, options *CopyOptions, out io.Writer) error {
	if options.Source == "" {
		return fmt.Errorf("source is required")
	}
	if options.Dest == "" {
		return fmt.Errorf("dest is required")
	}

	layerStore, err := factory.LayerStore()
	if err != nil {
		return err
	}

	destTokens := strings.SplitN(options.Dest, ":", 2)
	if len(destTokens) != 2 {
		return fmt.Errorf("unknown dest %q - expected <layer>:<path>", options.Dest)
	}

	l, err := layerStore.FindLayer(destTokens[0])
	if err != nil {
		return err
	}

	if l == nil {
		return fmt.Errorf("layer %q does not exist", destTokens[0])
	}

	if err := putFile(options.Source, l, destTokens[1], 0); err != nil {
		return err
	}
	fmt.Fprintf(out, "Copied %s -> %s\n", options.Source, options.Dest)
	return nil
}

func putFile(src string, l layers.Layer, dest string, depth int) error {
	stat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error reading %q: %v", src, err)
	}

	if stat.IsDir() {
		files, err := ioutil.ReadDir(src)
		if err != nil {
			return fmt.Errorf("error reading directory: %v", err)
		}

		if depth == 0 {
			// TODO: Mirror cp's magic (but tricky) syntax?
			// dest = filepath.Join(dest, stat.Name())
		}

		for _, file := range files {
			err := putFile(filepath.Join(src, file.Name()), l, filepath.Join(dest, file.Name()), depth+1)
			if err != nil {
				return err
			}
		}

		// Create the dirs, for the timestamps primarily
		// Note we have to do this after we write the files!
		// TODO: If we later copy another file into the directory, that will also change the modtime
		_, err = l.PutFile(dest, stat, nil)
		if err != nil {
			return err
		}

		return nil
	}

	glog.V(2).Infof("copying file %q to %s %q", src, l.Name(), dest)

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error reading %q: %v", src, err)
	}
	defer f.Close()

	_, err = l.PutFile(dest, stat, f)
	if err != nil {
		return err
	}

	return nil
}
