package main

/*
type SimpleFileData struct {
	path string
	size int64
}

func NewSimpleFileData(path string, size int64) *SimpleFileData {
	f := &SimpleFileData{path: path, size: size}
	return f
}

var _ FileData = &SimpleFileData{}

func (f *SimpleFileData) Size() (int64, error) {
	return f.size, nil
}

func (f *SimpleFileData) WriteTo(w io.Writer) (int64, error) {
	in, err := os.Open(f.path)
	if err != nil {
		return 0, err
	}
	defer func() {
		err := in.Close()
		if err != nil {
			glog.Warning("error closing source file (%s): %v", f.path, err)
		}
	}()

	n, err := io.Copy(w, in)
	if err != nil {
		return n, fmt.Errorf("error copying source file (%s): %v", f.path, err)
	}
	return n, nil
}
*/
