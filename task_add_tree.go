package main

import (
	"fmt"
	"io/ioutil"
	"strings"
)

type AddTreeTask struct {
	SourcePath string
	DestPath   []string
}

func ensureTrailingSlash(p string) string {
	if !strings.HasSuffix(p, "/") {
		p = p + "/"
	}
	return p
}

func (t *AddTreeTask) Run(b *BuildContext) error {
	sourceRoot := ensureTrailingSlash(t.SourcePath)

	destPath := t.DestPath
	err := t.walkTree(b, sourceRoot, destPath)
	return err
}

// Walk a tree, copying files
// We don't use ioutil.WalkTree so we can pass dest path around easily
func (t *AddTreeTask) walkTree(b *BuildContext, sourcePath string, dest []string) error {
	entries, err := ioutil.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("error reading source directory (%s): %v", sourcePath, err)
	}

	childDest := make([]string, len(dest))
	copy(childDest, dest)

	for _, entry := range entries {
		name := entry.Name()
		childDest[len(dest)] = name

		if entry.IsDir() {
			childName := ensureTrailingSlash(sourcePath + name)
			err = t.walkTree(b, childName, childDest)
			if err != nil {
				return err
			}
		} else {
			replace := false
			err = b.Layer.AddFileEntry(childDest, sourcePath+name, entry, replace)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
