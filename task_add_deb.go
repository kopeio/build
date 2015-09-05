package main

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/blakesmith/ar"
	"github.com/golang/glog"
)

type AddDebTask struct {
	Source   ByteSource
	DestPath []string
}

func (t *AddDebTask) Run(b *BuildContext) error {
	in, err := t.Source.Open()
	if err != nil {
		return fmt.Errorf("error reading source deb (%s): %v", t.Source.Name(), err)
	}
	defer loggedClose(in, t.Source.Name())

	reader := ar.NewReader(in)

	foundData := false

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("error reading deb file (%s): %v", t.Source.Name(), err)
		}

		glog.Infof("ar entry: %s", header.Name)

		if header.Name == "data.tar.xz" || header.Name == "data.tar.xz/" {
			foundData = true
			xz := &XZByteSource{Inner: NewOnceByteSource(ioutil.NopCloser(reader), t.Source.Name())}
			tarTask := AddTarTask{Source: xz, DestPath: t.DestPath}
			err = tarTask.Run(b)
			if err != nil {
				return err
			}
		} else if header.Name == "data.tar.gz" || header.Name == "data.tar.gz/" {
			foundData = true
			src := &GZIPByteSource{Inner: NewOnceByteSource(ioutil.NopCloser(reader), t.Source.Name())}
			tarTask := AddTarTask{Source: src, DestPath: t.DestPath}
			err = tarTask.Run(b)
			if err != nil {
				return err
			}
		} else {
			_, err = io.Copy(ioutil.Discard, reader)
			if err != nil {
				return fmt.Errorf("error reading deb file entry (%s): %v", t.Source.Name(), err)
			}
		}
	}

	if !foundData {
		return fmt.Errorf("unable to find data segment in %s", t.Source.Name())
	}

	return nil
}
