package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"kope.io/build/pkg/docker"
	"kope.io/build/pkg/layers"
)

type FetchOptions struct {
	Source string
}

func BuildFetchCommand(f Factory, out io.Writer) *cobra.Command {
	options := &FetchOptions{}

	cmd := &cobra.Command{
		Use: "fetch",
		Run: func(cmd *cobra.Command, args []string) {
			options.Source = cmd.Flags().Arg(0)
			if err := RunFetchCommand(f, options, out); err != nil {
				ExitWithError(err)
			}
		},
	}

	return cmd
}

func RunFetchCommand(factory Factory, options *FetchOptions, out io.Writer) error {
	if options.Source == "" {
		return fmt.Errorf("source is required")
	}

	layerStore, err := factory.LayerStore()
	if err != nil {
		return err
	}

	spec, err := ParseDockerImageSpec(options.Source)
	if err != nil {
		return err
	}

	glog.Infof("Querying registry for image %s", spec)

	registry := &docker.Registry{
		URL: spec.Host,
	}
	auth := &docker.Auth{}

	{
		dockerManifest, err := registry.GetManifest(auth, spec.Repository, spec.Tag)
		if err != nil {
			return fmt.Errorf("error getting manifest: %v", err)
		}
		glog.Infof("response: %s", dockerManifest)

		blob, err := ensureBlob(out, registry, auth, spec.Repository, dockerManifest.Config.Digest, dockerManifest.Config.Size, layerStore)
		if err != nil {
			return err
		}

		manifest := &layers.ImageManifest{
			Repository: spec.Repository,
			Tag:        spec.Tag,
			Config: layers.LayerManifest{
				Digest: blob.Digest(),
				Size:   blob.Length(),
			},
		}

		for _, layer := range dockerManifest.Layers {
			blob, err := ensureBlob(out, registry, auth, spec.Repository, layer.Digest, layer.Size, layerStore)
			if err != nil {
				return err
			}

			manifest.Layers = append(manifest.Layers, layers.LayerManifest{
				Digest: blob.Digest(),
				Size:   blob.Length(),
			})
		}

		if err := layerStore.WriteImageManifest(spec.Repository, spec.Tag, manifest); err != nil {
			return fmt.Errorf("error storing image manifest: %v", err)
		}
	}

	fmt.Fprintf(out, "Fetched %s\n", options.Source)
	return nil
}

func ensureBlob(out io.Writer, registry *docker.Registry, auth *docker.Auth, repository string, digest string, size int64, layerStore layers.Store) (layers.Blob, error) {
	blob, err := layerStore.FindBlob(repository, digest)
	if err != nil {
		return nil, fmt.Errorf("error checking for blob: %v", err)
	}
	if blob != nil {
		glog.Infof("already have blob %s", digest)
		return blob, nil
	}
	mb := size / (1024 * 1024)
	fmt.Fprintf(out, "Downloading layer %s (%d MB)\n", digest, mb)

	tmpfile, err := ioutil.TempFile("", "blob")
	if err != nil {
		return nil, fmt.Errorf("error creating temp file: %v", err)
	}
	defer func() {
		err := tmpfile.Close()
		if err != nil {
			glog.Warningf("error closing temp file %q: %v", tmpfile.Name(), err)
		}
		err = os.Remove(tmpfile.Name())
		if err != nil {
			glog.Warningf("error removing temp file %q: %v", tmpfile.Name(), err)
		}
	}()

	n, err := registry.DownloadBlob(auth, repository, digest, tmpfile)
	if err != nil {
		return nil, fmt.Errorf("error downloading blob %s: %v", digest, err)
	}

	glog.V(2).Infof("Downloaded blob %s size=%d", digest, n)

	_, err = tmpfile.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("error seeking to start of temp file: %v", err)
	}

	blob, err = layerStore.AddBlob(repository, digest, tmpfile)
	if err != nil {
		return nil, fmt.Errorf("error adding layer blob: %v", err)
	}
	return blob, nil
}
