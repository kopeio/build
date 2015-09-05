package main

import (
	"io"

	"github.com/golang/glog"
)

type hookedCloser struct {
	io.Reader

	inner io.Closer
}

func (c *hookedCloser) Close() error {
	return c.inner.Close()
}

func HookedCloser(i io.Reader, c io.Closer) io.ReadCloser {
	return &hookedCloser{Reader: i, inner: c}
}

func loggedClose(c io.Closer, name string) {
	err := c.Close()
	if err != nil {
		if name != "" {
			glog.Warningf("error closing %s: %v", name, err)
		} else {
			glog.Warningf("error closing %v: %v", c, err)
		}
	}
}
