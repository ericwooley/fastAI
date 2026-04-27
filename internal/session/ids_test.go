package session

import (
	"errors"
	"testing"
)

func TestValidateSessionID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id      string
		wantErr bool
	}{
		{id: "follow-up_1"},
		{id: "session.name-2"},
		{id: "" , wantErr: true},
		{id: "..", wantErr: true},
		{id: "bad/path", wantErr: true},
		{id: "bad\\path", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			err := ValidateSessionID(tt.id)
			if tt.wantErr && !errors.Is(err, ErrInvalidSessionID) {
				t.Fatalf("expected invalid session id, got %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRepoKeyStable(t *testing.T) {
	t.Parallel()
	key1, err := RepoKey(".")
	if err != nil {
		t.Fatalf("repo key: %v", err)
	}
	key2, err := RepoKey("./")
	if err != nil {
		t.Fatalf("repo key: %v", err)
	}
	if key1 != key2 || key1 == "" {
		t.Fatalf("repo key should be stable: %q %q", key1, key2)
	}
}
