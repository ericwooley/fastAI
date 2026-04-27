package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cli/oauth"
	"github.com/cli/oauth/device"
)

const CopilotClientID = "Ov23li8tweQw6odWQebz"

type postFormer interface {
	PostForm(string, url.Values) (*http.Response, error)
}

type Authenticator interface {
	Login(context.Context, io.Writer) (Account, error)
}

type DeviceFlowAuthenticator struct {
	ClientID   string
	Scopes     []string
	Host       *oauth.Host
	HTTPClient postFormer
	Now        func() time.Time
}

func NewDeviceFlowAuthenticator() (*DeviceFlowAuthenticator, error) {
	host, err := oauth.NewGitHubHost("https://github.com")
	if err != nil {
		return nil, err
	}
	return &DeviceFlowAuthenticator{
		ClientID:   CopilotClientID,
		Scopes:     []string{"read:user"},
		Host:       host,
		HTTPClient: http.DefaultClient,
		Now:        time.Now,
	}, nil
}

func (a *DeviceFlowAuthenticator) Login(ctx context.Context, out io.Writer) (Account, error) {
	clientID := a.ClientID
	if clientID == "" {
		clientID = CopilotClientID
	}
	scopes := a.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read:user"}
	}
	host := a.Host
	if host == nil {
		var err error
		host, err = oauth.NewGitHubHost("https://github.com")
		if err != nil {
			return Account{}, err
		}
	}
	client := a.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	now := a.Now
	if now == nil {
		now = time.Now
	}

	code, err := device.RequestCode(client, host.DeviceCodeURL, clientID, scopes)
	if err != nil {
		return Account{}, err
	}
	if out != nil {
		fmt.Fprintf(out, "Open this URL to authenticate with GitHub: %s\n", code.VerificationURI)
		fmt.Fprintf(out, "Enter code: %s\n", code.UserCode)
	}
	token, err := device.Wait(ctx, client, host.TokenURL, device.WaitOptions{ClientID: clientID, DeviceCode: code})
	if err != nil {
		return Account{}, err
	}
	scopes = strings.Fields(token.Scope)
	if len(scopes) == 0 {
		scopes = a.Scopes
	}
	return Account{
		Provider:        ProviderGitHubCopilot,
		AccessToken:     token.Token,
		Scopes:          scopes,
		LastValidatedAt: now(),
	}, nil
}

type StaticAuthenticator struct {
	Account Account
	Err     error
}

func (a StaticAuthenticator) Login(context.Context, io.Writer) (Account, error) {
	if a.Err != nil {
		return Account{}, a.Err
	}
	return a.Account, nil
}
