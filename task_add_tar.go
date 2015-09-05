package main

import (
	"archive/tar"
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
)

type AddTarTask struct {
	Source   ByteSource
	DestPath []string
}

func (t *AddTarTask) Run(b *BuildContext) error {
	name := t.Source.Name()
	in, err := t.Source.Open()
	if err != nil {
		return fmt.Errorf("error opening tar (%s): %v", name, err)
	}

	defer loggedClose(in, name)

	reader := tar.NewReader(in)

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("error reading tar file (%s): %v", name, err)
		}

		glog.Infof("tar entry: %s", header.Name)

		var content ByteSource
		isDir := false
		includeContent := true
		switch header.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			includeContent = true

		case tar.TypeDir:
			includeContent = false
			isDir = true

		default:
			glog.Warning("Unknown tar entry type: ", header.Typeflag)
		}

		if includeContent {
			buffered, err := NewBufferByteSource(reader, name)
			if err != nil {
				return fmt.Errorf("error reading tar file entry (%s): %v", name, err)
			}
			content = buffered
		}

		srcPath := strings.Split(header.Name, "/")
		if len(srcPath) != 0 && srcPath[0] == "." {
			srcPath = srcPath[1:]
		}
		if len(srcPath) != 0 && srcPath[len(srcPath)-1] == "" {
			srcPath = srcPath[0 : len(srcPath)-1]
		}

		write := true
		if len(srcPath) == 0 {
			write = false
		}

		if write {
			destPath := make([]string, 0, len(t.DestPath)+len(srcPath))
			destPath = append(destPath, t.DestPath...)
			destPath = append(destPath, srcPath...)

			if isDir {
				err = b.Layer.Mkdirp(destPath, header)
				if err != nil {
					return fmt.Errorf("error adding dir to image (%s): %v", name, err)
				}
			} else {
				replace := false
				err = b.Layer.AddEntry(destPath, content, header, replace)
				if err != nil {
					return fmt.Errorf("error adding file to image (%s): %v", name, err)
				}
			}
		}
	}
	return nil
}
