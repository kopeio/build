package docker

import (
	"net/http"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"github.com/golang/glog"
	"io"
	"strconv"
	"bytes"
	"crypto/rand"
	"strings"
	"net/url"
)

type Registry struct {
	URL string
}

type tagListResponse struct {
	Name string `json:"name"`
	Tags []string `json:"tags"`
}

func (r *Registry) ListTags(token *Token, image string) ([]string, error) {
	client := &http.Client{
	}

	authHeader, err := token.GetAuthorizationHeader()
	if err != nil {
		return nil, err
	}

	url := "https://registry-1.docker.io/v2/" + image + "/tags/list"
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

//type ManifestV1Layer struct {
//	BlobSum string `json:"blobSum"`
//}
//
//type ManifestV struct {
//	SchemaVersion int `json:"schemaVersion"`
//	Name          string `json:"name"`
//	Tag           string `json:"tag"`
//	Architecture  string `json:"architecture"`
//	FSLayers      []ManifestV1Layer `json:"fslayers"`
//}
//"history": [
//{
//"v1Compatibility": "{\"architecture\":\"amd64\",\"config\":{\"Hostname\":\"1295ff10ed92\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Cmd\":[\"sh\"],\"Image\":\"sha256:0d7e86beb406ca2ff3418fa5db5e25dd6f60fe7265d68a9a141a2aed005b1ae7\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":null,\"OnBuild\":null,\"Labels\":{}},\"container\":\"d12e9fb4928df60ac71b4b47d56b9b6aec383cccceb3b9275029959403ab4f73\",\"container_config\":{\"Hostname\":\"1295ff10ed92\",\"Domainname\":\"\",\"User\":\"\",\"AttachStdin\":false,\"AttachStdout\":false,\"AttachStderr\":false,\"Tty\":false,\"OpenStdin\":false,\"StdinOnce\":false,\"Env\":[\"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"],\"Cmd\":[\"/bin/sh\",\"-c\",\"#(nop) \",\"CMD [\\\"sh\\\"]\"],\"Image\":\"sha256:0d7e86beb406ca2ff3418fa5db5e25dd6f60fe7265d68a9a141a2aed005b1ae7\",\"Volumes\":null,\"WorkingDir\":\"\",\"Entrypoint\":null,\"OnBuild\":null,\"Labels\":{}},\"created\":\"2017-03-09T18:28:04.586987216Z\",\"docker_version\":\"1.12.6\",\"id\":\"d7f9d170aa714bf287eb994aada10a57f4fb1bf8fd748cbfa36a7ab9549190c1\",\"os\":\"linux\",\"parent\":\"d0fb090f7b4758531261295c34b956bc0b8a6663fb98605612d3eb47a02cd7f5\",\"throwaway\":true}"
//},
//{
//"v1Compatibility": "{\"id\":\"d0fb090f7b4758531261295c34b956bc0b8a6663fb98605612d3eb47a02cd7f5\",\"created\":\"2017-03-09T18:28:03.975884948Z\",\"container_config\":{\"Cmd\":[\"/bin/sh -c #(nop) ADD file:c9ecd8ff00c653fb652ad5a0a9215e1f467f0cd9933653b8a2e5e475b68597ab in / \"]}}"
//}
//],
//"signatures": [
//{
//"header": {
//"jwk": {
//"crv": "P-256",
//"kid": "6JLX:PQSL:2EMG:WH2Z:Q74M:MGJO:HOEC:LTAQ:2CRD:FLEQ:HGTS:WCT3",
//"kty": "EC",
//"x": "_GLuuvcq2gBtHMuL96jAOSlux7a9_ghxqSbWkLHkN9g",
//"y": "jKwuTJa5npR7IqP6F_tQ_E6DXjIC00UBL2zpj7vn7ME"
//},
//"alg": "ES256"
//},
//"signature": "1vJjx9bWVQPIOeFuX5cvhHuN1DLNsf7kqsGEW1z7QucB4TJtoJ-TRaGuc9o3YTroIYTqz_W3cskECd52BEBFUw",
//"protected": "eyJmb3JtYXRMZW5ndGgiOjIwOTYsImZvcm1hdFRhaWwiOiJDbjAiLCJ0aW1lIjoiMjAxNy0wNC0yOFQwNDoyNTo0NVoifQ"
//}
//]


type ManifestV2Layer struct {
	MediaType string `json:"mediaType"`
	Size      int64 `json:"size"`
	Digest    string `json:"digest"`
}

type ManifestV2 struct {
	SchemaVersion int `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	//Name string `json:"name"`
	//Tag string `json:"tag"`
	//Architecture string `json:"architecture"`
	Layers        []ManifestV2Layer `json:"layers"`
	Config        ManifestV2Layer `json:"config"`
}

func (m *ManifestV2) String() string {
	v, err := json.Marshal(m)
	if err != nil {
		glog.V(2).Infof("error serializing %v", err)
		return "<error>"
	}
	return string(v)
}

func (r *Registry) GetManifest(token *Token, image string, tag string) (*ManifestV2, error) {
	client := &http.Client{
	}

	authHeader, err := token.GetAuthorizationHeader()
	if err != nil {
		return nil, err
	}

	url := "https://registry-1.docker.io/v2/" + image + "/manifests/" + tag
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting %q: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	glog.Infof("got docker manifest %s", body)
	response := &ManifestV2{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

func (r *Registry) PutManifest(token *Token, image string, tag string, manifest *ManifestV2) (error) {
	client := &http.Client{
	}

	authHeader, err := token.GetAuthorizationHeader()
	if err != nil {
		return err
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("error serializing manifest: %v", err)
	}


	url := "https://registry-1.docker.io/v2/" + image + "/manifests/" + tag
	glog.V(4).Infof("Creating manifest at %s: %s", url, string(data))

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting %q: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return  fmt.Errorf("error reading reply: %v", err)
	}

	if resp.StatusCode != 201 {
		glog.V(2).Infof("body %s", string(body))
		return fmt.Errorf("unexpected error writing image to docker registry: %s", resp.Status)
	}

	return nil
}

func (r *Registry) DownloadBlob(token *Token, repository string, digest string, w io.Writer) (int64, error) {
	client := &http.Client{
	}

	authHeader, err := token.GetAuthorizationHeader()
	if err != nil {
		return 0, err
	}

	url := "https://registry-1.docker.io/v2/" + repository + "/blobs/" + digest
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", authHeader)
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error getting %q: %v", url, err)
	}
	defer resp.Body.Close()

	return io.Copy(w, resp.Body)
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


func (r *Registry) monolithicUpload(token *Token, repository string, digest string, src io.Reader, length int64) (error) {
	client := &http.Client{
	}

	authHeader, err := token.GetAuthorizationHeader()
	if err != nil {
		return err
	}

	url := "https://registry-1.docker.io/v2/" + repository + "/blobs/uploads/?digest=" + digest
	req, err := http.NewRequest("POST", url, src)

	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Length", strconv.FormatInt(length, 10))
	req.Header.Add("Content-Type", "application/octet-stream")

	glog.V(2).Infof("Blob upload of size %d: %s %s", length, req.Method, req.URL)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error uploading %q: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("docker registry returned unexpected result uploading to %s: %s", repository, resp.Status)
	}
	return nil
}


func (r *Registry) beginUpload(token *Token, repository string) (string, error) {
	client := &http.Client{
	}

	authHeader, err := token.GetAuthorizationHeader()
	if err != nil {
		return "", err
	}

	url := "https://registry-1.docker.io/v2/" + repository + "/blobs/uploads/"
	req, err := http.NewRequest("POST", url, bytes.NewReader(nil))

	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Length", "0")

	glog.V(2).Infof("Initiate blob upload: %s %s", req.Method, req.URL)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error initiating blob upload %q: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return "", fmt.Errorf("docker registry returned unexpected result initiating upload to %s: %s", repository, resp.Status)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("no location returned from upload begin")
	}
	return location, nil
}


func (r *Registry) completeUpload(token *Token, location string, digest string, src io.Reader, length int64) (error) {
	client := &http.Client{
	}

	authHeader, err := token.GetAuthorizationHeader()
	if err != nil {
		return err
	}

	u := location
	if strings.Contains(u, "?") {
		u += "&"
	} else {
		u += "?"
	}
	u += "digest=" + url.QueryEscape(digest)
	req, err := http.NewRequest("PUT", u, src)

	req.Header.Add("Authorization", authHeader)
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

func (r *Registry) UploadBlob(token *Token, repository string, digest string, src io.Reader, length int64) (error) {
	// TODO: Chunked / resumable uploading?
	//return r.monolithicUpload(token, repository, digest, src, length)

	location, err := r.beginUpload(token, repository)
	if err != nil {
		return err
	}
	
	err = r.completeUpload(token, location, digest, src, length)
	if err != nil {
		return err
	}
	
	return nil
}

func (r *Registry) HasBlob(token *Token, repository string, digest string) (bool, error) {
	client := &http.Client{
	}

	authHeader, err := token.GetAuthorizationHeader()
	if err != nil {
		return false, err
	}

	url := "https://registry-1.docker.io/v2/" + repository + "/blobs/" + digest

	glog.V(2).Infof("Checking for blob: HEAD %s", url)
	req, err := http.NewRequest("HEAD", url, nil)
	req.Header.Add("Authorization", authHeader)
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error getting %q: %v", url, err)
	}
	defer resp.Body.Close()

	glog.V(2).Infof("Status = %s", resp.Status)

	if resp.StatusCode == 200 {
		return true, nil
	}
	if resp.StatusCode == 404 {
		return false, nil
	}
	if resp.StatusCode == 401 {
		wwwAuthenticateHeader := resp.Header.Get("www-authenticate")
		glog.Infof("Authentication needed: %s", wwwAuthenticateHeader)
	}

	return false, fmt.Errorf("unexpected response code checking for blob %s in %s: %s", digest, repository, resp.Status)
}