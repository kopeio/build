package docker

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

type Registry struct {
	URL string

	HttpClient *http.Client
}

type tagListResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func (r *Registry) ListTags(token *Token, image string) ([]string, error) {
	client := &http.Client{}

	authHeader, err := token.GetAuthorizationHeader()
	if err != nil {
		return nil, err
	}

	url := r.buildUrl("v2/" + image + "/tags/list")
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting %q: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	response := &tagListResponse{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Tags, nil
}

type ManifestV2Layer struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

type ManifestV2 struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	//Name string `json:"name"`
	//Tag string `json:"tag"`
	//Architecture string `json:"architecture"`
	Layers []ManifestV2Layer `json:"layers"`
	Config ManifestV2Layer   `json:"config"`
}

func (m *ManifestV2) String() string {
	v, err := json.Marshal(m)
	if err != nil {
		glog.V(2).Infof("error serializing %v", err)
		return "<error>"
	}
	return string(v)
}

func (r *Registry) GetManifest(auth *Auth, repository string, tag string) (*ManifestV2, error) {
	authHeader := auth.FindHeader(r, repository, "pull")

	attempt := 0
	for {
		attempt++

		url := r.buildUrl("v2/" + repository + "/manifests/" + tag)
		glog.V(4).Infof("Reading manifest at %s", url)

		req, err := http.NewRequest("GET", url, nil)
		if authHeader != "" {
			req.Header.Add("Authorization", authHeader)
		}
		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")

		resp, body, err := r.doSimpleRequest(req)
		if err != nil {
			return nil, fmt.Errorf("error writing manifest to %s: %v", repository, err)
		}

		switch resp.StatusCode {
		case 200:
			glog.V(4).Infof("got docker manifest %s", body)
			response := &ManifestV2{}
			err = json.Unmarshal(body, response)
			if err != nil {
				return nil, fmt.Errorf("error parsing response: %v", err)
			}
			return response, nil

		case 401:
			if attempt >= 2 {
				return nil, fmt.Errorf("permission denied reading %s", r.buildHumanName(repository, tag))
			}

			authHeader, err = auth.GetHeader(r, resp)
			if err != nil {
				return nil, err
			}

		default:
			glog.V(2).Infof("unexpected http response: %s %s", resp.Status, body)
			return nil, fmt.Errorf("docker registry returned unexpected result reading manifest %s: %s", repository, resp.Status)
		}
	}
}

func (r *Registry) PutManifest(auth *Auth, repository string, tag string, manifest *ManifestV2) error {
	authHeader := auth.FindHeader(r, repository, "pull,push")

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("error serializing manifest: %v", err)
	}

	attempt := 0
	for {
		attempt++

		url := r.buildUrl("v2/" + repository + "/manifests/" + tag)
		glog.V(4).Infof("Creating manifest at %s: %s", url, string(data))

		req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
		if authHeader != "" {
			req.Header.Add("Authorization", authHeader)
		}
		req.Header.Add("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")

		resp, body, err := r.doSimpleRequest(req)
		if err != nil {
			return fmt.Errorf("error writing manifest to %s: %v", repository, err)
		}

		switch resp.StatusCode {
		case 201:
			return nil

		case 401:
			if attempt >= 2 {
				return fmt.Errorf("permission denied uploading to %s", r.buildHumanName(repository, tag))
			}

			authHeader, err = auth.GetHeader(r, resp)
			if err != nil {
				return err
			}

		default:
			glog.V(2).Infof("unexpected http response: %s %s", resp.Status, body)
			return fmt.Errorf("docker registry returned unexpected result writing manifest to %s: %s", repository, resp.Status)
		}
	}

}

func (r *Registry) DownloadBlob(auth *Auth, repository string, digest string, w io.Writer) (int64, error) {
	authHeader := auth.FindHeader(r, repository, "pull")

	attempt := 0
	for {
		attempt++

		url := r.buildUrl("v2/" + repository + "/blobs/" + digest)
		glog.V(4).Infof("Reading blob at %s", url)

		req, err := http.NewRequest("GET", url, nil)
		if authHeader != "" {
			req.Header.Add("Authorization", authHeader)
		}
		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")

		httpClient := r.httpClient()

		glog.V(2).Infof("HTTP %s %s", req.Method, req.URL)
		resp, err := httpClient.Do(req)
		if err != nil {
			return 0, err
		}
		// We can't defer resp.Body.Close, because we're in a loop

		switch resp.StatusCode {
		case 200:
			n, err := io.Copy(w, resp.Body)

			resp.Body.Close()

			return n, err

		case 401:
			resp.Body.Close()

			if attempt >= 2 {
				return 0, fmt.Errorf("permission denied reading from %s", r.buildHumanName(repository, ""))
			}

			authHeader, err = auth.GetHeader(r, resp)
			if err != nil {
				return 0, err
			}

		default:
			resp.Body.Close()

			glog.V(2).Infof("unexpected http response: %s", resp.Status)
			return 0, fmt.Errorf("docker registry returned unexpected result reading blob %s/%s: %s", repository, digest, resp.Status)
		}
	}
}

// randomId returns a random string; it isn't technically a UUID
func randomId() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		glog.Fatalf("error generating random data: %v", err)
	}

	uuid := fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return uuid
}

func (r *Registry) httpClient() *http.Client {
	if r.HttpClient == nil {
		return http.DefaultClient
	}
	return r.HttpClient
}

func (r *Registry) buildUrl(relativePath string) string {
	url := r.URL
	if url == "" {
		url = "https://registry-1.docker.io/"
	}
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += relativePath
	return url
}

func (r *Registry) buildHumanName(repository, tag string) string {
	s := r.URL
	if repository != "" {
		if s != "" && !strings.HasSuffix(s, "/") {
			s += "/"
		}
		s += repository

		if tag != "" {
			s += ":" + tag
		}
	}
	return s
}

func (r *Registry) doSimpleRequest(req *http.Request) (*http.Response, []byte, error) {
	httpClient := r.httpClient()

	glog.V(2).Infof("HTTP %s %s", req.Method, req.URL)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading response body: %v")
	}

	return resp, body, err
}

func (r *Registry) beginUpload(auth *Auth, repository string) (string, string, error) {
	authHeader := auth.FindHeader(r, repository, "pull,push")

	attempt := 0

	for {
		attempt++

		url := r.buildUrl("v2/" + repository + "/blobs/uploads/")

		req, err := http.NewRequest("POST", url, bytes.NewReader(nil))
		if authHeader != "" {
			req.Header.Add("Authorization", authHeader)
		}
		req.Header.Add("Content-Length", "0")

		glog.V(2).Infof("Initiate blob upload: %s %s", req.Method, req.URL)
		resp, _, err := r.doSimpleRequest(req)
		if err != nil {
			return "", "", fmt.Errorf("error initiating upload to %s: %v", repository, err)
		}

		switch resp.StatusCode {
		case 202:
			location := resp.Header.Get("Location")
			if location == "" {
				return "", "", fmt.Errorf("no location returned from upload begin")
			}
			return authHeader, location, nil

		case 401:
			if attempt >= 2 {
				return "", "", fmt.Errorf("permission denied uploading to %s", r.buildHumanName(repository, ""))
			}

			authHeader, err = auth.GetHeader(r, resp)
			if err != nil {
				return "", "", err
			}

		default:
			return "", "", fmt.Errorf("docker registry returned unexpected result initiating upload to %s: %s", repository, resp.Status)
		}
	}
}

func (r *Registry) completeUpload(authHeader string, location string, digest string, src io.Reader, length int64) error {
	client := r.httpClient()

	u := location
	if strings.Contains(u, "?") {
		u += "&"
	} else {
		u += "?"
	}
	u += "digest=" + url.QueryEscape(digest)
	req, err := http.NewRequest("PUT", u, src)

	if authHeader != "" {
		req.Header.Add("Authorization", authHeader)
	}
	req.Header.Add("Content-Length", strconv.FormatInt(length, 10))
	req.Header.Add("Content-Type", "application/octet-stream")

	glog.V(2).Infof("Blob upload of size %d: %s %s", length, req.Method, req.URL)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error uploading %q: %v", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("docker registry returned unexpected result uploading to %s: %s", u, resp.Status)
	}

	actualDigest := resp.Header.Get("docker-content-digest")
	if actualDigest != digest {
		return fmt.Errorf("upload to docker registry returned different digest %q from expected %q", actualDigest, digest)
	}
	return nil
}

// monolithic upload is in the spec but not actually supported?  https://github.com/docker/distribution/issues/1170
//func (r *Registry) monolithicUpload(auth *Auth, repository string, digest string, src io.Reader, length int64) error {
//	authHeader := auth.FindHeader(r, repository, "pull,push")
//
//	attempt := 0
//
//	for {
//		attempt++
//
//		url := r.buildUrl("v2/" + repository + "/blobs/uploads/?digest=" + url.QueryEscape(digest))
//
//		req, err := http.NewRequest("POST", url, src)
//		if authHeader != "" {
//			req.Header.Add("Authorization", authHeader)
//		}
//		req.Header.Add("Content-Length", strconv.FormatInt(length, 10))
//		req.Header.Add("Content-Type", "application/octet-stream")
//
//		glog.V(2).Infof("Initiate blob upload: %s %s", req.Method, req.URL)
//		resp, _, err := r.doSimpleRequest(req)
//		if err != nil {
//			return fmt.Errorf("error initiating upload to %s: %v", repository, err)
//		}
//
//		switch (resp.StatusCode) {
//		case 202:
//			return nil
//
//		case 401:
//			if attempt >= 2 {
//				return fmt.Errorf("permission denied")
//			}
//
//			authHeader, err = auth.GetHeader(r, resp)
//			if err != nil {
//				return err
//			}
//
//		default:
//			return fmt.Errorf("docker registry returned unexpected result initiating upload to %s: %s", repository, resp.Status)
//		}
//	}
//}

func (r *Registry) UploadBlob(auth *Auth, repository string, digest string, src io.Reader, length int64) error {
	// TODO: Chunked / resumable uploading?
	authHeader, location, err := r.beginUpload(auth, repository)
	if err != nil {
		return err
	}

	err = r.completeUpload(authHeader, location, digest, src, length)
	if err != nil {
		return err
	}

	return nil
}

func (r *Registry) HasBlob(auth *Auth, repository string, digest string) (bool, error) {
	authHeader := auth.FindHeader(r, repository, "pull")

	attempt := 0

	for {
		attempt++

		url := r.buildUrl("v2/" + repository + "/blobs/" + digest)

		req, err := http.NewRequest("HEAD", url, bytes.NewReader(nil))
		if authHeader != "" {
			req.Header.Add("Authorization", authHeader)
		}

		glog.V(2).Infof("Checking for blob: HEAD %s", url)
		resp, _, err := r.doSimpleRequest(req)
		if err != nil {
			return false, fmt.Errorf("error checking for blob %s/%s: %v", repository, digest, err)
		}

		switch resp.StatusCode {
		case 200:
			return true, nil

		case 404:
			return false, nil

		case 401:
			if attempt >= 2 {
				return false, fmt.Errorf("permission denied reading blob from %s", r.buildHumanName(repository, ""))
			}

			authHeader, err = auth.GetHeader(r, resp)
			if err != nil {
				return false, err
			}

		default:
			return false, fmt.Errorf("unexpected response code checking for blob %s/%s: %s", repository, digest, resp.Status)
		}
	}
}
