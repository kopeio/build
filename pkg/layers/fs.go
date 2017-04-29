package layers

import (
	"path/filepath"
	"fmt"
	"os"
	"io"
	"strings"
	"encoding/json"
	"io/ioutil"
	"github.com/golang/glog"
	"archive/tar"
	"path"
	"crypto/sha256"
	"encoding/hex"
	"compress/gzip"
)

type FSLayerStore struct {
	Path string
}

var _ Store = &FSLayerStore{}

func (s *FSLayerStore) CreateLayer(name string, options Options) (Layer, error) {
	p := filepath.Join(s.Path, "layers", name)
	err := os.MkdirAll(p, 0755)
	if err != nil {
		return nil, fmt.Errorf("error creating layer directory %q: %v", p, err)
	}
	l := &fsLayer{
		name: name,
		path: p,
	}
	if err = l.SetOptions(options); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *FSLayerStore) FindLayer(name string) (Layer, error) {
	p := filepath.Join(s.Path, "layers", name)
	_, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading layer directory %q: %v", p, err)
	}
	l := &fsLayer{
		name: name,
		path: p,
	}
	return l, nil
}

func (s *FSLayerStore) DeleteLayer(name string) (error) {
	p := filepath.Join(s.Path, "layers", name)
	_, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("error reading layer directory %q: %v", p, err)
	}

	err = os.RemoveAll(p)
	if err != nil {
		return fmt.Errorf("error deleting layer directory %q: %v", p, err)
	}

	return nil
}

func (s *FSLayerStore) AddBlob(repository string, digest string, r io.Reader) (Blob, error) {
	p := filepath.Join(s.Path, "blob", repository, digest)
	err := os.MkdirAll(filepath.Dir(p), 0755)
	if err != nil {
		return nil, fmt.Errorf("error creating blob directory %q: %v", p, err)
	}

	actualDigest, n, err := putFileWithSha(p, 0644, r)
	if err != nil {
		return nil, err
	}
	glog.Infof("copied blob to %q", p)
	if digest != actualDigest {
		err := os.Remove(p)
		if err != nil {
			glog.Warningf("error removing blob with bad digest %q: %v", p, err)
		}
		glog.Fatalf("digest does not match: %q vs %q", digest, actualDigest)
	}
	return &FSBlob{
		p: p,
		digest: digest,
		length: n,
		repository: repository,
	}, nil
}

func (s *FSLayerStore) FindBlob(repository string, digest string) (Blob, error) {
	if digest == "" {
		return nil, fmt.Errorf("digest is required")
	}
	p := filepath.Join(s.Path, "blob", repository, digest)
	glog.V(4).Infof("checking for blob %s", p)
	stat, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(4).Infof("blob file not found %q", p)
			return nil, nil
		}
		return nil, fmt.Errorf("error reading blob file%q: %v", p, err)
	}

	return &FSBlob{
		p: p,
		digest: digest,
		length: stat.Size(),
		repository: repository,
	}, nil
}

type FSBlob struct {
	store *FSLayerStore
	p          string
	digest     string
	repository string
	length int64
}

func (b *FSBlob) Digest() string {
	return b.digest
}

func (b *FSBlob) Length() int64{
	return b.length
}

func (b *FSBlob) Open() (io.ReadCloser, error) {
	return os.Open(b.p)
}

func (s *FSLayerStore) WriteImageManifest(repository string, tag string, manifest *ImageManifest) error {
	p := filepath.Join(s.Path, "image", repository, tag)
	err := os.MkdirAll(filepath.Dir(p), 0755)
	if err != nil {
		return fmt.Errorf("error creating manifest directory %q: %v", p, err)
	}

	manifest.Repository = repository
	manifest.Tag = tag

	dataJson, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing data: %v", err)
	}

	err = ioutil.WriteFile(p, dataJson, 0644)
	if err != nil {
		return fmt.Errorf("error writing data %q: %v", p, err)
	}

	return nil
}

func (s *FSLayerStore) FindImageManifest(repository string, tag string) (*ImageManifest, error) {
	p := filepath.Join(s.Path, "image", repository, tag)

	glog.V(4).Infof("reading image manifest %q", p)

	b, err := ioutil.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		} else {
			return nil, fmt.Errorf("error reading image metadata %q: %v", p, err)
		}
	}

	meta := &ImageManifest{}
	err = json.Unmarshal(b, meta)
	if err != nil {
		return nil, fmt.Errorf("error parsing image metadata %q: %v", p, err)
	}

	return meta, nil
}

type fsLayer struct {
	path string
	name string
}

func (l *fsLayer) Name() string {
	return l.name
}

func (l *fsLayer) PutFile(dest string, stat os.FileInfo, in io.Reader) (int64, error) {
	dest = strings.TrimPrefix(dest, "/")
	// TODO: Sanitize to remove .. etc

	dest = filepath.Join(l.path, "rootfs", dest)
	err := os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return 0, fmt.Errorf("failed to mkdirs for %q: %v", dest, err)
	}

	return putFile(dest, stat.Mode(), in)
}

func putFile(dest string, mode os.FileMode, in io.Reader) (n int64, err error) {
	out, err := os.OpenFile(dest, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, mode)
	if err != nil {
		err = fmt.Errorf("failed to write %q: %v", dest, err)
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	n, err = io.Copy(out, in)
	return
}

func putFileWithSha(dest string, mode os.FileMode, in io.Reader) (hash string, n int64, err error) {
	out, err := os.OpenFile(dest, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, mode)
	if err != nil {
		err = fmt.Errorf("failed to write %q: %v", dest, err)
		return
	}

	hasher := sha256.New()
	mw := io.MultiWriter(out, hasher)
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	n, err = io.Copy(mw, in)
	if err != nil {
		return
	}

	hash = "sha256:" + hex.EncodeToString(hasher.Sum(nil))

	return
}

type layerMetadata struct {
	Options Options `json:"options"`
}

func (f *fsLayer )SetOptions(options Options) error {
	meta, err := f.readMetadata()
	if err != nil {
		return err
	}
	meta.Options = options

	return f.writeMetadata(meta)
}

func (f *fsLayer )GetOptions() (Options, error) {
	var options Options
	meta, err := f.readMetadata()
	if err != nil {
		return options, err
	}
	options = meta.Options
	return options, nil
}

func (f *fsLayer )readMetadata() (*layerMetadata, error) {
	p := filepath.Join(f.path, "metadata.json")
	metaJson, err := ioutil.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			metaJson = nil
		} else {
			return nil, fmt.Errorf("error reading layer metadata %q: %v", p, err)
		}
	}

	meta := &layerMetadata{}
	if len(metaJson) != 0 {
		err = json.Unmarshal(metaJson, meta)
		if err != nil {
			return nil, fmt.Errorf("error parsing layer metadata %q: %v", p, err)
		}
	}

	return meta, nil
}

func (l *fsLayer) writeMetadata(meta *layerMetadata) (error) {
	p := filepath.Join(l.path, "metadata.json")
	metaJson, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing metadata: %v", err)
	}

	err = ioutil.WriteFile(p, metaJson, 0644)
	if err != nil {
		return fmt.Errorf("error writing metadata %q: %v", p, err)
	}

	return nil
}

func (l *fsLayer) BuildTar(store Store, repository string) (Blob, string, error) {
	tmpfile, err := ioutil.TempFile("", "layer")
	if err != nil {
		return nil, "", fmt.Errorf("error creating temp file: %v", err)
	}

	defer func() {
		err := tmpfile.Close()
		if err != nil {
			glog.Warningf("error closing temp file %q: %v", tmpfile.Name(), err)
		}
		err = os.Remove(tmpfile.Name())
		if err != nil {
			glog.Warningf("error removing temp file %q: %v", tmpfile.Name(), err)
		}
	}()

	hasher := sha256.New()
	mw := io.MultiWriter(hasher, tmpfile)

	gzipWriter := gzip.NewWriter(mw)
	defer func() {
		if gzipWriter != nil {
			gzipWriter.Close()
		}
	}()

	hasherUncompressed := sha256.New()
	mwUncompressed := io.MultiWriter(hasherUncompressed, gzipWriter)

	w := tar.NewWriter(mwUncompressed)
	defer func() {
		if w != nil {
			w.Close()
		}
	}()

	rootfs := filepath.Join(l.path, "rootfs")
	err = copyDirToTar(w, "", nil, rootfs)
	if err != nil {
		 return nil, "", fmt.Errorf("error building tar: %v", err)
	}

	err = w.Close()
	if err != nil {
		return nil, "", fmt.Errorf("error closing tar: %v", err)
	}

	// Avoid double-closing tar
	w = nil


	err = gzipWriter.Close()
	if err != nil {
		return nil, "", fmt.Errorf("error closing gzip writer: %v", err)
	}

	// Avoid double-closing gzip
	gzipWriter = nil

	digest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))

	_, err = tmpfile.Seek(0, 0)
	if err != nil {
		return nil, "", fmt.Errorf("error seeking to start of temp file: %v", err)
	}

	// TODO: move?

	blob, err := store.AddBlob(repository, digest, tmpfile)
	if err != nil {
		return nil, "", fmt.Errorf("error storing blob: %v", err)
	}

	diffID := "sha256:" + hex.EncodeToString(hasherUncompressed.Sum(nil))

	return blob, diffID, nil

}

func copyDirToTar(w *tar.Writer, tarPrefix string, f os.FileInfo, srcDir string) error {
	if tarPrefix != "" {
		hdr, err := tar.FileInfoHeader(f, "")
		if err != nil {
			return fmt.Errorf("error build tar entry: %v", err)
		}
		hdr.Name = path.Join(tarPrefix, f.Name())

		err = w.WriteHeader(hdr)
		if err != nil {
			return fmt.Errorf("error creating tar entry for directory %s: %v", hdr.Name, err)
		}
	}

	files, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("error reading directory %q: %v", srcDir, err)
	}

	for _, f := range files {
		if f.IsDir() {
			err = copyDirToTar(w, tarPrefix + f.Name() + "/", f, filepath.Join(srcDir, f.Name()))
			if err != nil {
				return err

			}
			continue
		}

		err = copyFileToTar(w, tarPrefix, f, srcDir)
		if err != nil {
			return err
		}
	}

	return nil
}


func copyFileToTar(w *tar.Writer, tarPrefix string, f os.FileInfo, srcDir string) error {
	hdr, err := tar.FileInfoHeader(f, "")
	if err != nil {
		return fmt.Errorf("error build tar entry: %v", err)
	}
	hdr.Name = path.Join(tarPrefix, f.Name())

	err = w.WriteHeader(hdr)
	if err != nil {
		return fmt.Errorf("error creating tar entry: %v", err)
	}

	p := filepath.Join(srcDir, f.Name())
	in, err := os.Open(p)
	if err != nil {
		return err
	}
	defer in.Close()

	_, err = io.Copy(w, in)
	if err != nil {
		return fmt.Errorf("error copying file %q to tarfile: %v", p, err)
	}

	return nil
}