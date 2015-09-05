package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/glog"
)

type DockerBundle struct {
	Layers []*Layer
}

func NewDockerBundle() *DockerBundle {
	b := &DockerBundle{}
	return b
}
func (b *DockerBundle) WriteToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file (%s): %v", path, err)
	}
	defer loggedClose(f, path)

	return b.Write(f)
}

func (b *DockerBundle) Write(w io.Writer) error {
	tarWriter := tar.NewWriter(w)
	defer loggedClose(tarWriter, "tar-writer")

	for _, layer := range b.Layers {
		err := b.writeLayer(tarWriter, "", layer)
		if err != nil {
			return fmt.Errorf("error writing layer (%s): %v", layer.ID, err)
		}
	}

	// TODO: Write tags?
	return nil
}

func writeTarDir(w *tar.Writer, name string) error {
	header := buildDirectoryTarHeader(name, 0755)
	err := w.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("error writing tar header for dir (%s): %v", name, err)
	}
	return nil
}

func buildDirectoryTarHeader(name string, mode int) *tar.Header {
	if !strings.HasSuffix(name, "/") {
		name = name + "/"
	}
	header := &tar.Header{}
	header.Mode = 0755
	header.Typeflag = tar.TypeDir
	header.Name = name
	return header
}
func writeBytesAsTarFile(w *tar.Writer, name string, contents []byte) error {
	buf := bytes.NewBuffer(contents)
	return writeTarFile(w, name, buf, int64(len(contents)))
}

func writeTarFile(w *tar.Writer, name string, src io.Reader, length int64) error {
	header := &tar.Header{}
	header.Mode = 0644
	header.Typeflag = tar.TypeReg
	header.Name = name
	header.Size = length

	err := w.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("error writing tar header for file (%s): %v", name, err)
	}
	n, err := io.Copy(w, src)
	if err != nil {
		return fmt.Errorf("error writing tar contents for file (%s): %v", name, err)
	}

	if n != length {
		return fmt.Errorf("wrong length when writing tar file %s: %v", name, err)
	}

	return nil
}

func (b *DockerBundle) writeLayer(w *tar.Writer, prefix string, layer *Layer) error {
	err := writeTarDir(w, prefix+layer.ID)
	if err != nil {
		return err
	}

	err = writeBytesAsTarFile(w, prefix+layer.ID+"/VERSION", []byte("1.0"))
	if err != nil {
		return err
	}

	layerJson, err := layer.BuildDockerJSON()
	if err != nil {
		return err
	}

	err = writeBytesAsTarFile(w, prefix+layer.ID+"/json", []byte(layerJson))
	if err != nil {
		return err
	}

	layerTempfile, err := ioutil.TempFile("", "layer")
	if err != nil {
		return fmt.Errorf("error creating temp file: %v", err)
	}

	defer func() {
		err := os.Remove(layerTempfile.Name())
		if err != nil {
			glog.Warningf("error removing temp file (%s): %v", layerTempfile.Name(), err)
		}
	}()

	tarWriter := tar.NewWriter(layerTempfile)
	defer loggedClose(tarWriter, "tarwriter")

	err = layer.WriteTar(tarWriter)
	if err != nil {
		return err
	}

	err = tarWriter.Flush()
	if err != nil {
		return fmt.Errorf("error writing layer tar file: %v", err)
	}

	layerTempfileSize, err := layerTempfile.Seek(0, 1)
	if err != nil {
		return fmt.Errorf("error getting position in temp file: %v", err)
	}

	_, err = layerTempfile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("error rewinding temp file: %v", err)
	}

	err = writeTarFile(w, prefix+layer.ID+"/layer.tar", layerTempfile, layerTempfileSize)
	if err != nil {
		return err
	}

	return nil
}
