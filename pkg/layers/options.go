package layers

type Options struct {
	WorkingDir string
	Cmd        []string
	Env        map[string]string

	Base string
}
