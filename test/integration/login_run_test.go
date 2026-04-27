package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/auth"
	"github.com/ericwooley/fastAI/internal/cli"
	appsession "github.com/ericwooley/fastAI/internal/session"
)

func TestLoginAndAuthenticatedRunOrchestration(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	config := t.TempDir()
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	store := auth.NewFileStore(filepath.Join(config, "auth.json"))
	deps := cli.Dependencies{
		AuthStore:      store,
		Authenticator:  auth.StaticAuthenticator{Account: account()},
		SessionService: appsession.NewService(appsession.NewFileStore(filepath.Join(config, "sessions")), time.Now),
		Runner:         runner,
		RepoRoot:       repo,
		Now:            time.Now,
	}

	if code := cli.Execute(context.Background(), []string{"login"}, deps); code != int(cli.ExitSuccess) {
		t.Fatalf("login exit = %d", code)
	}
	if _, err := store.Load(context.Background()); err != nil {
		t.Fatalf("load account after login: %v", err)
	}
	if code := cli.Execute(context.Background(), []string{"--model", "github:gpt-4.1", "do", "work"}, deps); code != int(cli.ExitSuccess) {
		t.Fatalf("run exit = %d", code)
	}
	if runner.seen.Model != "github:gpt-4.1" || runner.seen.AccessToken != "token" || !strings.Contains(runner.seen.Prompt, "do work") {
		t.Fatalf("runner saw unexpected request: %+v", runner.seen)
	}
}

func TestDeviceFlowLoginUsesGitHubOAuthShape(t *testing.T) {
	t.Parallel()
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/login/device/code" {
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				return nil, err
			}
			if req.Header.Get("Accept") != "application/json" || req.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("unexpected device headers: %v", req.Header)
			}
			if body["client_id"] != auth.CopilotClientID || body["scope"] != "read:user" {
				t.Fatalf("unexpected device body: %v", body)
			}
			return response(200, `{"device_code":"dev","user_code":"ABC","verification_uri":"https://github.com/login/device","expires_in":1,"interval":0}`), nil
		}
		if req.URL.Path == "/login/oauth/access_token" {
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				return nil, err
			}
			if body["client_id"] != auth.CopilotClientID || body["device_code"] != "dev" || body["grant_type"] != "urn:ietf:params:oauth:grant-type:device_code" {
				t.Fatalf("unexpected token body: %v", body)
			}
			return response(200, `{"access_token":"token","token_type":"bearer","scope":"read:user"}`), nil
		}
		return response(404, `{}`), nil
	})}
	host, err := oauthHostForTest("https://github.com")
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	authenticator := &auth.DeviceFlowAuthenticator{ClientID: auth.CopilotClientID, Host: host, HTTPClient: client, Now: time.Now}
	account, err := authenticator.Login(context.Background(), nil)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if account.AccessToken != "token" {
		t.Fatalf("token = %q", account.AccessToken)
	}
	if account.OAuthClientID != auth.CopilotClientID {
		t.Fatalf("oauth client id = %q", account.OAuthClientID)
	}
	if strings.Join(account.Scopes, ",") != "read:user" {
		t.Fatalf("scopes = %v", account.Scopes)
	}
}
