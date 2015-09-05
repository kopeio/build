package main

import (
	"compress/gzip"
	"fmt"
	"io"
)

type GZIPByteSource struct {
	Inner ByteSource
}

func (b *GZIPByteSource) Name() string {
	return b.Inner.Name()
}

func (b *GZIPByteSource) Open() (io.ReadCloser, error) {
	f, err := b.Inner.Open()
	if err != nil {
		return nil, err
	}

	r, err := gzip.NewReader(f)
	if err != nil {
		loggedClose(f, b.Name())
		return nil, fmt.Errorf("error opening file for GZIP decompression (%s): %v", b.Name(), err)
	}
	return HookedCloser(r, f), nil
}
