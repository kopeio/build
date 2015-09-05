package main

import (
	"io"
	"os"
)

type FileByteSource struct {
	Path string
}

func NewFileByteSource(path string) *FileByteSource {
	return &FileByteSource{Path: path}
}

var _ ByteSource = &FileByteSource{}

func (f *FileByteSource) Open() (io.ReadCloser, error) {
	return os.Open(f.Path)
}

func (f *FileByteSource) Name() string {
	return f.Path
}
