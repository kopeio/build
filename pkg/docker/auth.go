package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Auth struct {
	HttpClient *http.Client
	URL        string
	Subject    string

	cache []*Token
}

type Token struct {
	auth   *Auth
	bearer *tokenResponse
	basic  string

	registry string
	scopes   []string
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

func (a *Auth) FindHeader(registry *Registry, repository string, permission string) string {
	token := a.findToken(registry, repository, permission)
	if token == nil {
		return ""
	}

	header, err := token.GetAuthorizationHeader()
	if err != nil {
		glog.Infof("error getting header from token: %v", err)
	}

	return header
}

func (a *Auth) findToken(registry *Registry, repository string, permission string) *Token {
	for _, token := range a.cache {
		if token.registry != registry.URL {
			continue
		}

		if token.basic != "" {
			return token
		}

		if token.bearer != nil {
			scope := "repository:" + repository + ":" + permission
			for _, s := range token.scopes {
				if s == scope {
					return token
				}
			}
		}
	}
	glog.V(4).Infof("Could not find cached token for %s %s", repository, permission)
	return nil
}

func tokenizeWWWAuthenticate(s string) []string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case c == '"':
			lastQuote = c
			return false
		default:
			return c == ','
		}
	}

	return strings.FieldsFunc(s, f)
}

func (a *Auth) GetHeader(registry *Registry, resp *http.Response) (string, error) {
	wwwAuthenticate := resp.Header.Get("www-authenticate")
	if wwwAuthenticate == "" {
		return "", fmt.Errorf("Permission error, but did not recieve www-authenticate header")
	}

	if strings.HasPrefix(wwwAuthenticate, "Bearer ") {
		realm := ""
		service := ""
		scope := ""

		v := strings.TrimPrefix(wwwAuthenticate, "Bearer ")
		tokens := tokenizeWWWAuthenticate(v)
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

		token, err := a.getToken(registry, service, scope, realm)
		if err != nil {
			return "", err
		}

		if token == nil {
			glog.Infof("No authentication information for %q", realm)
			return "", nil
		}

		return token.GetAuthorizationHeader()

	} else if strings.HasPrefix(wwwAuthenticate, "Basic ") {
		// "Basic realm=\"basic-realm\""

		site := registry.URL
		authConfig, err := a.getAuthentication(site)
		if err != nil {
			return "", err
		}

		if authConfig == nil {
			glog.Infof("No authentication information for %q", site)
			return "", nil
		}

		token := &Token{
			auth:  a,
			basic: authConfig.Auth,

			registry: registry.URL,
		}
		a.cache = append(a.cache, token)
		return token.GetAuthorizationHeader()
	} else {
		return "", fmt.Errorf("unknown www-authenticate challenge: %q", wwwAuthenticate)
	}
}

func (a *Auth) getAuthentication(site string) (*authConfig, error) {
	var errors []error

	config := os.Getenv("REGISTRY_CONFIG")
	if config != "" {
		auth, err := getAuthentication([]byte(config), site)
		if err != nil {
			errors = append(errors, fmt.Errorf("error parsing REGISTRY_CONFIG: %v", err))
		} else if auth != nil {
			glog.Infof("Found credentials for %s in REGISTRY_CONFIG", site)
			return auth, nil
		}
	}

	{
		p := filepath.Join(os.Getenv("HOME"), ".dockercfg")
		b, err := ioutil.ReadFile(p)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("error reading %q: %v", p, err)
			} else {
				b = nil
			}
		}
		auth, err := getAuthentication(b, site)
		if err != nil {
			errors = append(errors, fmt.Errorf("error parsing %q: %v", p, err))
		} else if auth != nil {
			glog.Infof("Found credentials for %s in %q", site, p)
			return auth, nil
		}
	}

	if len(errors) == 1 {
		return nil, errors[0]
	} else if len(errors) > 1 {
		b := &bytes.Buffer{}
		for _, err := range errors {
			b.WriteString(fmt.Sprintf("  %v\n", err))
		}

		return nil, fmt.Errorf("Multiple errors getting authentication:\n%s", b.String())
	}

	glog.Infof("Did not find credentials for %s", site)
	return nil, nil
}

func getAuthentication(config []byte, site string) (*authConfig, error) {
	if len(config) == 0 {
		return nil, nil
	}

	conf := make(map[string]*authConfig)

	err := json.Unmarshal(config, &conf)
	if err != nil {
		return nil, fmt.Errorf("error parsing: %v", err)
	}

	var keys []string
	if site == "" || site == "https://registry-1.docker.io/" {
		keys = append(keys, "https://registry-1.docker.io/")
		keys = append(keys, "https://index.docker.io/")
		keys = append(keys, "https://index.docker.io/v1/")
	} else {
		keys = append(keys, site)
	}

	for _, k := range keys {
		auth := conf[k]
		if auth != nil {
			return auth, nil
		}
	}

	for _, k := range keys {
		k = strings.TrimPrefix(site, "https://")
		k = strings.TrimPrefix(k, "http://")
		auth := conf[k]
		if auth != nil {
			return auth, nil
		}
	}

	return nil, nil
}

func (a *Auth) getToken(registry *Registry, service string, scope string, realm string) (*Token, error) {
	httpClient := a.HttpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	site := registry.URL
	if site == "" {
		site = "https://registry-1.docker.io/"
	}
	glog.Infof("Requesting docker token for %s", site)

	auth, err := a.getAuthentication(site)
	if err != nil {
		return nil, err
	}

	// e.g. "https://auth.docker.io/token?service=registry.docker.io&scope=" + scope
	authUrl := realm
	var params []string
	if service != "" {
		params = append(params, "service="+url.QueryEscape(service))
	}
	if scope != "" {
		params = append(params, "scope="+url.QueryEscape(scope))
	}
	if len(params) != 0 {
		authUrl += "?" + strings.Join(params, "&")
	}
	req, err := http.NewRequest("GET", authUrl, nil)
	if auth != nil && auth.Auth != "" {
		req.Header.Add("authorization", "Basic "+auth.Auth)
	}
	glog.V(2).Infof("HTTP %s %s", req.Method, req.URL)

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

	glog.V(6).Infof("auth response %s", string(body))

	data := &tokenResponse{}
	err = json.Unmarshal(body, data)
	if err != nil {
		glog.V(2).Infof("bad response: %q", string(body))
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	// TODO split scope so repository:a:push,pull => repository:a:pull
	scopes := []string{scope}

	token := &Token{
		auth:     a,
		bearer:   data,
		registry: registry.URL,
		scopes:   scopes,
	}
	a.cache = append(a.cache, token)
	return token, nil
}

func (t *Token) GetAuthorizationHeader() (string, error) {
	if t.basic != "" {
		return "Basic " + t.basic, nil
	}

	if t.bearer != nil {
		// TODO: Check expiration and refresh
		return "Bearer " + t.bearer.Token, nil
	}

	return "", fmt.Errorf("invalid token")
}
