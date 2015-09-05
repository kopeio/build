package main

import (
	"fmt"
	"io"
)

type OnceByteSource struct {
	done  bool
	inner io.ReadCloser
	name  string
}

func NewOnceByteSource(inner io.ReadCloser, name string) *OnceByteSource {
	return &OnceByteSource{inner: inner, name: name}
}

var _ ByteSource = &OnceByteSource{}

func (f *OnceByteSource) Open() (io.ReadCloser, error) {
	if f.done {
		return nil, fmt.Errorf("OnceByteSource can only be read once")
	}
	f.done = true
	return f.inner, nil
}

func (f *OnceByteSource) Name() string {
	return f.name
}
