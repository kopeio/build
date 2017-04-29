package layers

import (
	"io"
	"os"
)

type Store interface {
	CreateLayer(name string, options Options) (Layer, error)
	DeleteLayer(name string) error
	FindLayer(name string) (Layer, error)

	AddBlob(repository string, digest string, src io.Reader) (Blob, error)
	FindBlob(repository string, digest string) (Blob, error)

	WriteImageManifest(repository string, tag string, manifest *ImageManifest) error
	FindImageManifest(repository string, tag string) (*ImageManifest, error)
}

type Layer interface {
	Name() string

	PutFile(dest string, stat os.FileInfo, r io.Reader) (int64, error)

	GetOptions() (Options, error)
	SetOptions(options Options) error

	BuildTar(destStore Store, destRepository string) (Blob, string, error)
}

type Blob interface {
	Digest() string
	Length() int64

	Open() (io.ReadCloser, error)
}

type ImageManifest struct {
	Repository string          `json:"repository"`
	Tag        string          `json:"tag"`
	Config     LayerManifest   `json:"config"`
	Layers     []LayerManifest `json:"layers"`
}

type LayerManifest struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}
