package main

import (
	"fmt"
	"io"

	"xi2.org/x/xz"
)

type XZByteSource struct {
	Inner ByteSource
}

func (l *XZByteSource) Name() string {
	return l.Inner.Name()
}

func (l *XZByteSource) Open() (io.ReadCloser, error) {
	f, err := l.Inner.Open()
	if err != nil {
		return nil, err
	}

	r, err := xz.NewReader(f, 0)
	if err != nil {
		loggedClose(f, l.Inner.Name())
		return nil, fmt.Errorf("error opening file for XZ decompression (%s): %v", l.Name(), err)
	}
	return HookedCloser(r, f), nil
}
