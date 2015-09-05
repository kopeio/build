package main

import (
	"archive/tar"
	crypto_rand "crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/golang/glog"
)

type BuildContext struct {
	Layer *Layer
}

func randomLayerID() string {
	b := make([]byte, 32)
	_, err := crypto_rand.Read(b)
	if err != nil {
		glog.Fatalf("error reading from random: %v", err)
	}
	s := hex.EncodeToString(b)
	return s
}

func NewBuildContext() *BuildContext {
	b := &BuildContext{}
	id := randomLayerID()
	b.Layer = NewLayer(id)
	return b
}

func (b *BuildContext) WriteImage(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file (%s): %v", path, err)
	}
	defer loggedClose(f, path)

	tarWriter := tar.NewWriter(f)
	defer loggedClose(tarWriter, "tar:"+path)

	return b.Layer.WriteTar(tarWriter)
}
