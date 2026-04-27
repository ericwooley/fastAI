package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ProviderGitHubCopilot = "github-copilot"

var (
	ErrNoAccount = errors.New("no authenticated account")
	ErrExpired   = errors.New("authenticated account expired")
	ErrClientID  = errors.New("authenticated account was created for a different OAuth client")
)

type Account struct {
	Provider        string     `json:"provider"`
	UserID          string     `json:"user_id,omitempty"`
	Login           string     `json:"login,omitempty"`
	AccessToken     string     `json:"access_token"`
	OAuthClientID   string     `json:"oauth_client_id,omitempty"`
	Scopes          []string   `json:"scopes,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	LastValidatedAt time.Time  `json:"last_validated_at"`
}

type Store interface {
	Save(context.Context, Account) error
	Load(context.Context) (Account, error)
}

type FileStore struct {
	path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Save(_ context.Context, account Account) error {
	if strings.TrimSpace(account.AccessToken) == "" {
		return fmt.Errorf("access token is required")
	}
	if account.Provider == "" {
		account.Provider = ProviderGitHubCopilot
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *FileStore) Load(_ context.Context) (Account, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Account{}, ErrNoAccount
		}
		return Account{}, err
	}
	var account Account
	if err := json.Unmarshal(data, &account); err != nil {
		return Account{}, err
	}
	if strings.TrimSpace(account.AccessToken) == "" {
		return Account{}, ErrNoAccount
	}
	return account, nil
}

func (a Account) Check(now time.Time) error {
	if strings.TrimSpace(a.AccessToken) == "" {
		return ErrNoAccount
	}
	if a.ExpiresAt != nil && !a.ExpiresAt.After(now) {
		return ErrExpired
	}
	return nil
}

func (a Account) CheckForClient(now time.Time, clientID string) error {
	if err := a.Check(now); err != nil {
		return err
	}
	if strings.TrimSpace(clientID) != "" && a.OAuthClientID != clientID {
		return ErrClientID
	}
	return nil
}
