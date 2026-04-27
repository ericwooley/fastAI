package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type RunRecord struct {
	RunID      string    `json:"run_id"`
	Prompt     string    `json:"prompt"`
	Model      string    `json:"model"`
	Outcome    string    `json:"outcome"`
	Summary    string    `json:"summary"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

type Record struct {
	SessionID  string      `json:"session_id"`
	RepoKey    string      `json:"repo_key"`
	Model      string      `json:"model"`
	Status     Status      `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	LastPrompt string      `json:"last_prompt"`
	LastRunID  string      `json:"last_run_id"`
	Runs       []RunRecord `json:"runs"`
}

type Store interface {
	Save(context.Context, Record) error
	Load(context.Context, string, string) (Record, error)
}

type FileStore struct {
	baseDir string
}

func DefaultConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "fastAI"), nil
}

func NewFileStore(baseDir string) *FileStore {
	return &FileStore{baseDir: baseDir}
}

func (s *FileStore) Save(_ context.Context, record Record) error {
	if err := ValidateSessionID(record.SessionID); err != nil {
		return err
	}
	if record.RepoKey == "" {
		return fmt.Errorf("repo key is required")
	}
	path := s.recordPath(record.RepoKey, record.SessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *FileStore) Load(_ context.Context, repoKey string, sessionID string) (Record, error) {
	if err := ValidateSessionID(sessionID); err != nil {
		return Record{}, err
	}
	data, err := os.ReadFile(s.recordPath(repoKey, sessionID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Record{}, ErrSessionNotFound
		}
		return Record{}, err
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return Record{}, fmt.Errorf("read session %q: %w", sessionID, err)
	}
	if record.RepoKey != repoKey {
		return Record{}, ErrRepoMismatch
	}
	return record, nil
}

func (s *FileStore) recordPath(repoKey string, sessionID string) string {
	return filepath.Join(s.baseDir, repoKey, sessionID+".json")
}
