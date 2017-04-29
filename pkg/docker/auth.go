package docker

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

type Auth struct {
	URL string
}

type Token struct {
	auth *Auth
	data *tokenResponse
}

type tokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

type authConfig struct {
	Email string `json:"email"`
	Auth  string `json:"auth"`
}

func (a *Auth) getAuthentication(site string) (*authConfig, error) {
	conf := make(map[string]*authConfig)

	p := filepath.Join(os.Getenv("HOME"), ".dockercfg")
	b, err := ioutil.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading %q: %v", p, err)
	}

	err = json.Unmarshal(b, &conf)
	if err != nil {
		return nil, fmt.Errorf("error parsing %q: %v", p, err)
	}

	auth := conf[site]
	if auth != nil {
		return auth, nil
	}

	return nil, nil
}

func (a *Auth) GetToken(scope string) (*Token, error) {
	client := &http.Client{}

	glog.Infof("Requesting docker token for %s", scope)

	site := "https://index.docker.io/v1/"
	auth, err := a.getAuthentication(site)
	if err != nil {
		return nil, err
	}

	authUrl := "https://auth.docker.io/token?service=registry.docker.io&scope=" + scope
	req, err := http.NewRequest("GET", authUrl, nil)
	//req.Header.Add("If-None-Match", `W/"wyzzy"`)
	if auth != nil && auth.Auth != "" {
		req.Header.Add("authorization", "Basic "+auth.Auth)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting auth token for %q: %v", scope, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error getting auth token for %q: %v", scope, err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error getting auth token for %q: %s", scope, resp.Status)
	}

	glog.Infof("response %s", string(body))

	data := &tokenResponse{}
	err = json.Unmarshal(body, data)
	if err != nil {
		glog.V(2).Infof("bad response: %q", string(body))
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return &Token{auth: a, data: data}, nil
}

func (t *Token) GetAuthorizationHeader() (string, error) {
	// TODO: Check expiration and refresh
	return "Bearer " + t.data.Token, nil
}
