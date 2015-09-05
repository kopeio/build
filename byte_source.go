package main

import "io"

type ByteSource interface {
	Open() (io.ReadCloser, error)
	Name() string
}
