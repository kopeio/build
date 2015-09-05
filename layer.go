package main

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
)

type Layer struct {
	ID     string
	Config DockerConfig
	root   LayerEntry
}

type DockerConfig struct {
	ID              string                 `json:"id"`
	Parent          string                 `json:"parent,omitempty"`
	Created         string                 `json:"created,omitempty"`
	Container       string                 `json:"container,omitempty"`
	ContainerConfig *DockerContainerConfig `json:"container_config,omitempty"`
	DockerVersion   string                 `json:"docker_version"`
	Config          DockerContainerConfig  `json:"config"`
	Architecture    string                 `json:"architecture"`
	OS              string                 `json:"os"`
	Size            int64                  `json:"Size"`
}

type DockerContainerConfig struct {
	Hostname     string
	Domainname   string
	User         string
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	//PortSpecs: null
	// ExposedPorts: null
	Tty       bool
	OpenStdin bool
	StdinOnce bool
	Env       []string
	Cmd       []string
	Image     string
	// Volumes null
	VolumeDriver string
	WorkingDir   string
	// Entrypoint null
	NetworkDisabled bool
	MacAddress      string
	OnBuild         []string
	Labels          map[string]string
}

func (l *Layer) BuildDockerJSON() (string, error) {
	data, err := json.Marshal(l.Config)
	if err != nil {
		return "", fmt.Errorf("unable to build Docker JSON descriptor: %v", err)
	}
	return string(data), nil
}

func NewLayer(id string) *Layer {
	l := &Layer{}
	l.ID = id
	l.root.root = true

	now := time.Now().UTC()

	l.Config.ID = l.ID
	l.Config.Created = now.Format(time.RFC3339Nano)
	l.Config.DockerVersion = "1.7.1"
	l.Config.Architecture = "amd64"
	l.Config.OS = "linux"

	return l
}

type LayerEntry struct {
	root     bool
	meta     tar.Header
	Data     ByteSource
	Children map[string]*LayerEntry
}

func (e *LayerEntry) Name() string {
	return e.meta.Name
}

func (e *LayerEntry) FindChild(name string) *LayerEntry {
	if e.Children == nil {
		return nil
	}
	child, found := e.Children[name]
	if !found {
		return nil
	}
	return child
}

func (l *Layer) FindEntry(path []string) *LayerEntry {
	pos := &l.root
	for i := 0; i < len(path); i++ {
		pos = pos.FindChild(path[i])
		if pos == nil {
			return nil
		}
	}
	return pos
}

func (l *Layer) Open(path []string) (io.ReadCloser, error) {
	entry := l.FindEntry(path)
	if entry == nil {
		return nil, fmt.Errorf("entry not found: %s", strings.Join(path, "/"))
	}
	if entry.Data == nil {
		return nil, fmt.Errorf("entry is not a file: %s", strings.Join(path, "/"))
	}
	return entry.Data.Open()
}

func (l *Layer) Exists(path []string) bool {
	entry := l.FindEntry(path)
	return entry != nil
}

func (l *Layer) Mkdirp(path []string, header *tar.Header) error {
	entry := l.FindEntry(path)
	if entry != nil {
		// TODO: Check if dir?
		return nil
	}

	err := l.AddEntry(path, nil, header, false)
	return err
}

func (l *Layer) AddEntry(path []string, source ByteSource, meta *tar.Header, replace bool) error {
	glog.V(2).Infof("AddEntry: %s", path)
	parent := l.FindEntry(path[:len(path)-1])
	if parent == nil {
		return fmt.Errorf("directory does not exist: %s", strings.Join(path, "/"))
	}

	filename := path[len(path)-1]
	if parent.Children == nil {
		parent.Children = make(map[string]*LayerEntry)
	}
	existing, _ := parent.Children[filename]
	if existing != nil {
		if !replace {
			return fmt.Errorf("entry already exists with name: %s", strings.Join(path, "/"))
		}
	}

	child := &LayerEntry{}
	child.meta = *meta
	child.meta.Name = strings.Join(path, "/")

	child.Data = source

	parent.Children[filename] = child

	return nil
}

func (l *Layer) AddFileEntry(path []string, srcPath string, srcInfo os.FileInfo, replace bool) error {
	symlinkTarget := ""
	if (srcInfo.Mode() & os.ModeSymlink) != 0 {
		// TODO
		panic("symlinks not implemented")
	}
	meta, err := tar.FileInfoHeader(srcInfo, symlinkTarget)
	if err != nil {
		return fmt.Errorf("error converting file metadata to tar (for %s): %v", srcPath, err)
	}

	var source ByteSource
	if !srcInfo.IsDir() {
		source = NewFileByteSource(srcPath)
	}

	/*

		meta.ModTime = srcInfo.ModTime()
		var source ByteSource
		if srcInfo.IsDir() {
			meta.Size = 0
		} else {
			source = NewFileByteSource(srcPath)
			meta.Size = srcInfo.Size()
		}

		stat := srcInfo.Sys().(*syscall.Stat_t)
		meta.Mode = int64(stat.Mode)
		meta.Uid = int(stat.Uid)
		meta.Gid = int(stat.Gid)
		// We skip access time!

		meta.ModTime = toTime(stat.Mtimespec)
	*/

	err = l.AddEntry(path, source, meta, replace)
	if err != nil {
		return err
	}
	return nil
}

func toTime(src syscall.Timespec) time.Time {
	sec, nsec := src.Unix()
	return time.Unix(sec, nsec)
}

func (i *Layer) WriteTar(w *tar.Writer) error {
	return i.root.WriteTar(w, "/")
}

func (l *Layer) DebugDump(w io.Writer, indent string) error {
	_, err := io.WriteString(w, indent+"Layer\n")
	if err != nil {
		return err
	}
	return l.root.DebugDump(w, indent+"  ")
}

func (e *LayerEntry) DebugDump(w io.Writer, indent string) error {
	_, err := io.WriteString(w, indent+e.Name()+"\n")
	if err != nil {
		return err
	}
	if e.Children != nil {
		for _, child := range e.Children {
			err = child.DebugDump(w, indent+"  ")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *LayerEntry) WriteTar(w *tar.Writer, path string) error {
	if !e.root {
		glog.V(2).Infof("Tar writing entry: %s", e.meta.Name)
		err := w.WriteHeader(&e.meta)
		if err != nil {
			return fmt.Errorf("error writing tar entry header (for %s): %v", path, err)
		}

		if e.Data != nil {
			in, err := e.Data.Open()
			if err != nil {
				return fmt.Errorf("error writing tar entry contents (for %s): %v", path, err)
			}

			defer loggedClose(in, e.Data.Name())
			n, err := io.Copy(w, in)
			if err != nil {
				return fmt.Errorf("error writing tar entry contents (for %s): %v", path, err)
			}

			if n != e.meta.Size {
				return fmt.Errorf("error writing tar entry contents - file size mismatch (for %s): %v", path, err)
			}
		}
	}

	if len(e.Children) != 0 {
		for _, child := range e.Children {
			childPath := path + "/" + child.Name()
			err := child.WriteTar(w, childPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
