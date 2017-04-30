package docker

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"net/url"
)

type Auth struct {
	HttpClient *http.Client
	URL        string
	Subject    string
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

func (a *Auth) FindHeader(registry *Registry, repository string, scope string) (string) {
	glog.Warningf("Auth FindHeader not implemented")
	return ""
}

func (a *Auth) GetHeader(registry *Registry, resp *http.Response) (string, error) {
	wwwAuthenticate := resp.Header.Get("www-authenticate")
	if wwwAuthenticate == "" {
		return "", fmt.Errorf("Permission error, but did not recieve www-authenticate header")
	}
	glog.Infof("www-authenticate header is %q", wwwAuthenticate)

	if strings.HasPrefix(wwwAuthenticate, "Bearer ") {
		realm := ""
		service := ""
		scope := ""

		v := strings.TrimPrefix(wwwAuthenticate, "Bearer ")
		tokens := strings.Split(v, ",")
		for _, token := range tokens {
			if strings.HasPrefix(token, "realm=\"") {
				realm = strings.TrimPrefix(token, "realm=\"")
				realm = strings.TrimSuffix(realm, "\"")
			} else if strings.HasPrefix(token, "service=\"") {
				service = strings.TrimPrefix(token, "service=\"")
				service = strings.TrimSuffix(service, "\"")
			} else if strings.HasPrefix(token, "scope=\"") {
				scope = strings.TrimPrefix(token, "scope=\"")
				scope = strings.TrimSuffix(scope, "\"")
			} else {
				return "", fmt.Errorf("cannot parse www-authenticate header: %q", wwwAuthenticate)
			}
		}

		//if scope == "" {
		//	return "", fmt.Errorf("scope not specified in www-authenticate header: %q", wwwAuthenticate)
		//}
		//if service == "" {
		//	return "", fmt.Errorf("service not specified in www-authenticate header: %q", wwwAuthenticate)
		//}
		if realm == "" {
			return "", fmt.Errorf("realm not specified in www-authenticate header: %q", wwwAuthenticate)
		}

		site := registry.URL
		token, err := a.getToken(site, service, scope, realm)
		if err != nil {
			return "", err
		}

		return token.GetAuthorizationHeader()

	} else if strings.HasPrefix(wwwAuthenticate, "Basic ") {
		// "Basic realm=\"basic-realm\""

		site := registry.URL
		authConfig, err := a.getAuthentication(site)
		if err != nil {
			return "", err
		}

		return "Basic " + authConfig.Auth, nil
	} else {
		return "", fmt.Errorf("unknown www-authenticate challenge: %q", wwwAuthenticate)
	}
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
		glog.Infof("Found credentials for %s", site)
		return auth, nil
	}

	k := strings.TrimPrefix(site, "https://")
	k = strings.TrimPrefix(k, "http://")
	auth = conf[k]
	if auth != nil {
		glog.Infof("Found credentials for %s", site)
		return auth, nil
	}

	glog.Infof("Did not find credentials for %s", site)
	return nil, nil
}

func (a *Auth) getToken(site string, service string, scope string, realm string) (*Token, error) {
	httpClient := a.HttpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	glog.Infof("Requesting docker token for %s", site)

	//site := "https://index.docker.io/v1/"
	auth, err := a.getAuthentication(site)
	if err != nil {
		return nil, err
	}

	//authUrl := "https://auth.docker.io/token?service=registry.docker.io&scope=" + scope
	authUrl := realm
	var params []string
	if service != "" {
		params = append(params, "service=" + url.QueryEscape(service))
	}
	if scope != "" {
		params = append(params, "scope=" + url.QueryEscape(scope))
	}
	if len(params) != 0 {
		authUrl += "?" + strings.Join(params, "&")
	}
	req, err := http.NewRequest("GET", authUrl, nil)
	if auth != nil && auth.Auth != "" {
		req.Header.Add("authorization", "Basic " + auth.Auth)
	}
	resp, err := httpClient.Do(req)
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

	glog.V(4).Infof("auth response %s", string(body))

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
