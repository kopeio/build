package cmd

import (
	"github.com/spf13/cobra"
	"io"
	"fmt"
	"github.com/golang/glog"
	"kope.io/imagebuilder/pkg/docker"
	"kope.io/imagebuilder/pkg/layers"
	"io/ioutil"
	"os"
)

type FetchOptions struct {
	Source string
}

func BuildFetchCommand(f Factory, out io.Writer) *cobra.Command {
	options := &FetchOptions{}

	cmd := &cobra.Command{
		Use: "fetch",
		RunE: func(cmd*cobra.Command, args []string) error {
			options.Source = cmd.Flags().Arg(0)
			return RunFetchCommand(f, options, out)
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

	registry := &docker.Registry{}
	auth := docker.Auth{}
	token, err := auth.GetToken("repository:" + spec.Repository + ":pull")
	if err != nil {
		return fmt.Errorf("error getting registry token: %v", err)
	}
	//{
	//	response, err := registry.ListTags(token, image)
	//	if err != nil {
	//		return fmt.Errorf("error listing tags: %v", err)
	//	}
	//	glog.Infof("response: %s", response)
	//}

	{
		dockerManifest, err := registry.GetManifest(token, spec.Repository, spec.Tag)
		if err != nil {
			return fmt.Errorf("error getting manifest: %v", err)
		}
		glog.Infof("response: %s", dockerManifest)

		blob, err := ensureBlob(registry, token, spec.Repository, dockerManifest.Config.Digest, layerStore)
		if err != nil {
			return err
		}

		manifest := &layers.ImageManifest{
			Repository: spec.Repository,
			Tag: spec.Tag,
			Config: layers.LayerManifest{
				Digest: blob.Digest(),
				Size: blob.Length(),
			},
		}

		for _, layer := range dockerManifest.Layers {
			blob, err := ensureBlob(registry, token, spec.Repository, layer.Digest, layerStore)
			if err != nil {
				return err
			}

			manifest.Layers = append(manifest.Layers, layers.LayerManifest{
				Digest: blob.Digest(),
				Size: blob.Length(),
			})
		}

		if err := layerStore.WriteImageManifest(spec.Repository, spec.Tag, manifest); err != nil {
			return fmt.Errorf("error storing image manifest: %v", err)
		}
	}

	fmt.Fprintf(out, "Fetched %s\n", options.Source)
	return nil
}

func ensureBlob(registry *docker.Registry, token *docker.Token, repository string, digest string, layerStore layers.Store) (layers.Blob, error) {
	blob, err := layerStore.FindBlob(repository, digest)
	if err != nil {
		return nil, fmt.Errorf("error checking for blob: %v", err)
	}
	if blob != nil {
		glog.Infof("already have blob %s", digest)
		return blob, nil
	}
	glog.Infof("Downloading blob %s", digest)

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

	n, err := registry.DownloadBlob(token, repository, digest, tmpfile)
	if err != nil {
		return nil, fmt.Errorf("error downloading blob %s: %v", digest, err)
	}

	glog.Infof("Downloaded blob %s size=%d", digest, n)

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
