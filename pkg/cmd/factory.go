package cmd

import (
	"path/filepath"

	"kope.io/build/pkg/layers"
)

type Factory interface {
	LayerStore() (layers.Store, error)
}

type fsFactory struct {
	dir        string
	layerStore layers.Store
}

var _ Factory = &fsFactory{}

func newFSFactory(dir string) Factory {
	f := &fsFactory{
		dir: dir,
	}
	f.layerStore = &layers.FSLayerStore{
		Path: filepath.Join(dir),
	}
	return f
}

func (f *fsFactory) LayerStore() (layers.Store, error) {
	return f.layerStore, nil
}
