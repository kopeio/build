package docker

import (
	"reflect"
	"testing"
)

func TestGetAuthentication(t *testing.T) {
	input := `
	{
        "auths": {
			"https://index.docker.io/v1/": {
					"auth": "aGVsbG86d29ybGQ="
			}
		}
	}
`
	grid := []struct {
		Input    string
		Site     string
		Expected *dockerConfigAuth
	}{
		{
			Input:    input,
			Site:     "https://index.docker.io/v1/",
			Expected: &dockerConfigAuth{Auth: "aGVsbG86d29ybGQ="},
		},
		{
			Input:    input,
			Site:     "https://registry-1.docker.io/",
			Expected: &dockerConfigAuth{Auth: "aGVsbG86d29ybGQ="},
		},
		{
			Input:    input,
			Site:     "https://private.registry.io",
			Expected: nil,
		},
	}

	for _, g := range grid {
		actual, err := getAuthentication([]byte(g.Input), g.Site)
		if err != nil {
			t.Errorf("unexpected error getting authentication config: %v", err)
			continue
		}

		if !reflect.DeepEqual(actual, g.Expected) {
			t.Errorf("unexpected authConfig for %q: actual=%v expected=%v", g.Site, actual, g.Expected)
		}
	}

}
