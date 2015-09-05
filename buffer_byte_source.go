package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
)

// TODO: Don't always buffer in memory
type BufferByteSource struct {
	buffer bytes.Buffer
	name   string
}

func NewBufferByteSource(src io.Reader, name string) (*BufferByteSource, error) {
	b := &BufferByteSource{}
	b.name = name

	_, err := io.Copy(&b.buffer, src)
	if err != nil {
		return nil, fmt.Errorf("error buffering data (from %s): %v", name, err)
	}

	return b, nil
}

var _ ByteSource = &BufferByteSource{}

func (b *BufferByteSource) Open() (io.ReadCloser, error) {
	return ioutil.NopCloser(&b.buffer), nil
}

func (b *BufferByteSource) Name() string {
	return b.name
}
