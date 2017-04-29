package imageconfig

import (
	"kope.io/imagebuilder/pkg/layers"
	"runtime"
	"time"
)

type ImageConfig struct {
	Architecture    string          `json:"architecture"`
	Config          ContainerConfig `json:"config"`
	Container       string          `json:"container"`
	ContainerConfig ContainerConfig `json:"container_config"`
	Created         string          `json:"created"`
	DockerVersion   string          `json:"docker_version"`
	History         []History       `json:"history"`
	OS              string          `json:"os"`
	RootFS          RootFS          `json:"rootfs"`
}

type RootFS struct {
	Type    string   `json:"type"`
	DiffIDs []string `json:"diff_ids"`
}

type History struct {
	Created    string `json:"created"`
	CreatedBy  string `json:"created_by"`
	EmptyLayer bool   `json:"empty_layer,omitempty"`
}
type ContainerConfig struct {
	Hostname     string
	Domainname   string
	User         string
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	Tty          bool
	OpenStdin    bool
	StdinOnce    bool
	Env          []string
	Cmd          []string
	ArgsEscaped  bool
	Image        string
	Volumes      []string // ?
	WorkingDir   string
	Entrypoint   []string          // ?
	OnBuild      []string          // ?
	Labels       map[string]string // ?
}

func JoinLayer(base *ImageConfig, diffID string, digest string, description string, options layers.Options) (*ImageConfig, error) {
	c := &ImageConfig{}
	if base != nil {
		*c = *base
	} else {
		// TODO: It is really easy to do a crossbuild, so "same" might be an invalid assumption
		c.Architecture = runtime.GOARCH
		c.OS = runtime.GOOS
	}

	now := time.Now().UTC()
	c.Created = now.Format(time.RFC3339Nano)

	if options.WorkingDir != "" {
		c.Config.WorkingDir = options.WorkingDir
	}
	if options.Cmd != nil {
		c.Config.Cmd = options.Cmd
	}

	// TODO: Is this right?
	c.ContainerConfig = c.Config

	// History is ordered from base -> most derived
	c.History = append(c.History, History{
		Created:   now.Format(time.RFC3339Nano),
		CreatedBy: description,
	})

	// Layers are ordered from most derived -> base
	c.RootFS.Type = "layers"
	var layers []string
	for _, layer := range c.RootFS.DiffIDs {
		layers = append(layers, layer)
	}
	layers = append(layers, diffID)
	c.RootFS.DiffIDs = layers

	return c, nil
}
