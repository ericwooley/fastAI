package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreSaveLoadAndTokenState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewFileStore(filepath.Join(t.TempDir(), "auth.json"))
	if _, err := store.Load(ctx); !errors.Is(err, ErrNoAccount) {
		t.Fatalf("expected missing account, got %v", err)
	}

	expires := time.Now().Add(time.Hour)
	account := Account{AccessToken: "token", OAuthClientID: CopilotClientID, Login: "octo", ExpiresAt: &expires}
	if err := store.Save(ctx, account); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Provider != ProviderGitHubCopilot || loaded.AccessToken != "token" || loaded.OAuthClientID != CopilotClientID || loaded.Login != "octo" {
		t.Fatalf("unexpected account: %+v", loaded)
	}
	if err := loaded.Check(time.Now()); err != nil {
		t.Fatalf("check valid: %v", err)
	}
	if err := loaded.Check(expires.Add(time.Second)); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected expired, got %v", err)
	}
	if err := loaded.CheckForClient(time.Now(), "different-client"); !errors.Is(err, ErrClientID) {
		t.Fatalf("expected client mismatch, got %v", err)
	}
}
