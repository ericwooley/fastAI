package integration_test

import (
	"io"
	"net/http"
	"strings"

	"github.com/cli/oauth"
)

func oauthHostForTest(url string) (*oauth.Host, error) {
	return oauth.NewGitHubHost(url)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
