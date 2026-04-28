package session

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestServiceStartResumeAndRepoMatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := NewFileStore(filepath.Join(t.TempDir(), "sessions"))
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	service := NewService(store, func() time.Time { return now })
	repoA := makeRepo(t)
	repoB := makeRepo(t)

	record, resumed, err := service.Start(ctx, StartOptions{RepoRoot: repoA, Model: "m", Prompt: "first"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if resumed || record.SessionID == "" || record.Status != StatusActive {
		t.Fatalf("unexpected new session: %+v resumed=%v", record, resumed)
	}

	resumedRecord, resumed, err := service.Start(ctx, StartOptions{RepoRoot: repoA, SessionID: record.SessionID, Model: "m2", Prompt: "next"})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if !resumed || resumedRecord.Model != "m2" || resumedRecord.LastPrompt != "next" {
		t.Fatalf("unexpected resumed session: %+v resumed=%v", resumedRecord, resumed)
	}

	repoBRecord, resumed, err := service.Start(ctx, StartOptions{RepoRoot: repoB, SessionID: record.SessionID, Model: "m", Prompt: "bad"})
	if err != nil {
		t.Fatalf("start repo B explicit session: %v", err)
	}
	if resumed || repoBRecord.SessionID != record.SessionID || repoBRecord.RepoKey == record.RepoKey {
		t.Fatalf("expected new repo-scoped session, got %+v resumed=%v", repoBRecord, resumed)
	}
}

func makeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := osMkdir(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	return dir
}

var osMkdir = func(path string) error { return mkdirAll(path) }
