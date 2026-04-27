package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cli/oauth"
)

const (
	CopilotClientID          = "Ov23li8tweQw6odWQebz"
	defaultUserAgent         = "fastAI/0.1"
	oauthPollingSafetyMargin = 3 * time.Second
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Authenticator interface {
	Login(context.Context, io.Writer) (Account, error)
}

type DeviceFlowAuthenticator struct {
	ClientID   string
	Scopes     []string
	Host       *oauth.Host
	HTTPClient httpDoer
	UserAgent  string
	Now        func() time.Time
}

func NewDeviceFlowAuthenticator() (*DeviceFlowAuthenticator, error) {
	host, err := oauth.NewGitHubHost("https://github.com")
	if err != nil {
		return nil, err
	}
	return &DeviceFlowAuthenticator{
		ClientID:   CopilotClientID,
		Scopes:     defaultScopes(),
		Host:       host,
		HTTPClient: http.DefaultClient,
		UserAgent:  defaultUserAgent,
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
		scopes = defaultScopes()
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
	userAgent := a.UserAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	code, err := requestDeviceCode(ctx, client, host.DeviceCodeURL, clientID, scopes, userAgent)
	if err != nil {
		return Account{}, err
	}
	if out != nil {
		fmt.Fprintf(out, "Open this URL to authenticate with GitHub: %s\n", code.VerificationURI)
		fmt.Fprintf(out, "Enter code: %s\n", code.UserCode)
	}
	token, err := waitForToken(ctx, client, host.TokenURL, clientID, code, userAgent)
	if err != nil {
		return Account{}, err
	}
	scopes = strings.FieldsFunc(token.Scope, func(r rune) bool {
		return r == ' ' || r == ','
	})
	if len(scopes) == 0 {
		scopes = a.Scopes
	}
	return Account{
		Provider:        ProviderGitHubCopilot,
		AccessToken:     token.AccessToken,
		OAuthClientID:   clientID,
		Scopes:          scopes,
		LastValidatedAt: now(),
	}, nil
}

func defaultScopes() []string {
	return []string{"read:user"}
}

type deviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
	Error           string `json:"error"`
	ErrorDesc       string `json:"error_description"`
}

type accessToken struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
	Interval    int    `json:"interval"`
}

func requestDeviceCode(ctx context.Context, client httpDoer, url string, clientID string, scopes []string, userAgent string) (deviceCode, error) {
	payload := map[string]string{"client_id": clientID, "scope": strings.Join(scopes, " ")}
	body, err := json.Marshal(payload)
	if err != nil {
		return deviceCode{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return deviceCode{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return deviceCode{}, err
	}
	defer resp.Body.Close()
	var result deviceCode
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return deviceCode{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if result.Error != "" {
			return deviceCode{}, fmt.Errorf("GitHub device authorization failed: %s", result.Error)
		}
		return deviceCode{}, fmt.Errorf("GitHub device authorization failed: %s", resp.Status)
	}
	if result.DeviceCode == "" || result.UserCode == "" || result.VerificationURI == "" {
		return deviceCode{}, fmt.Errorf("GitHub device authorization response was incomplete")
	}
	return result, nil
}

func waitForToken(ctx context.Context, client httpDoer, url string, clientID string, code deviceCode, userAgent string) (accessToken, error) {
	interval := time.Duration(code.Interval) * time.Second
	if interval <= 0 {
		interval = time.Second
	}
	for {
		payload := map[string]string{
			"client_id":   clientID,
			"device_code": code.DeviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return accessToken{}, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return accessToken{}, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)
		resp, err := client.Do(req)
		if err != nil {
			return accessToken{}, err
		}
		var result accessToken
		decodeErr := json.NewDecoder(resp.Body).Decode(&result)
		_ = resp.Body.Close()
		if decodeErr != nil {
			return accessToken{}, decodeErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return accessToken{}, fmt.Errorf("GitHub OAuth token request failed: %s", resp.Status)
		}
		if result.AccessToken != "" {
			return result, nil
		}
		switch result.Error {
		case "authorization_pending":
			if err := sleep(ctx, interval+oauthPollingSafetyMargin); err != nil {
				return accessToken{}, err
			}
		case "slow_down":
			if result.Interval > 0 {
				interval = time.Duration(result.Interval) * time.Second
			} else {
				interval += 5 * time.Second
			}
			if err := sleep(ctx, interval+oauthPollingSafetyMargin); err != nil {
				return accessToken{}, err
			}
		case "":
			if err := sleep(ctx, interval+oauthPollingSafetyMargin); err != nil {
				return accessToken{}, err
			}
		default:
			if result.ErrorDesc != "" {
				return accessToken{}, fmt.Errorf("GitHub OAuth error: %s: %s", result.Error, result.ErrorDesc)
			}
			return accessToken{}, fmt.Errorf("GitHub OAuth error: %s", result.Error)
		}
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
