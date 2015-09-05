package main

import (
	"bytes"
	"io"
	"io/ioutil"
)

type SliceByteSource struct {
	data []byte
	name string
}

func NewSliceByteSource(src []byte, name string) *SliceByteSource {
	b := &SliceByteSource{}
	b.name = name
	b.data = src
	return b
}

var _ ByteSource = &SliceByteSource{}

func (b *SliceByteSource) Open() (io.ReadCloser, error) {
	buffer := bytes.NewBuffer(b.data)
	return ioutil.NopCloser(buffer), nil
}

func (b *SliceByteSource) Name() string {
	return b.name
}
