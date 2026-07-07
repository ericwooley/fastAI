package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Clock func() time.Time

type Service struct {
	store Store
	now   Clock
}

type StartOptions struct {
	RepoRoot  string
	SessionID string
	Model     string
	Prompt    string
}

func NewService(store Store, now Clock) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, now: now}
}

func (s *Service) Start(ctx context.Context, opts StartOptions) (Record, bool, error) {
	repoKey, err := RepoKey(opts.RepoRoot)
	if err != nil {
		return Record{}, false, err
	}
	now := s.now()
	if opts.SessionID != "" {
		if err := ValidateSessionID(opts.SessionID); err != nil {
			return Record{}, false, err
		}
		record, err := s.store.Load(ctx, repoKey, opts.SessionID)
		if err != nil {
			if err == ErrSessionNotFound {
				record = Record{
					SessionID:  opts.SessionID,
					RepoKey:    repoKey,
					Model:      opts.Model,
					Status:     StatusActive,
					CreatedAt:  now,
					UpdatedAt:  now,
					LastPrompt: opts.Prompt,
				}
				if err := s.store.Save(ctx, record); err != nil {
					return Record{}, false, err
				}
				return record, false, nil
			}
			return Record{}, false, err
		}
		record.Status = StatusActive
		record.Model = opts.Model
		record.LastPrompt = opts.Prompt
		record.UpdatedAt = now
		if err := s.store.Save(ctx, record); err != nil {
			return Record{}, false, err
		}
		return record, true, nil
	}

	record := Record{
		SessionID:  GenerateSessionID(),
		RepoKey:    repoKey,
		Model:      opts.Model,
		Status:     StatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastPrompt: opts.Prompt,
	}
	if err := s.store.Save(ctx, record); err != nil {
		return Record{}, false, err
	}
	return record, false, nil
}

func (s *Service) Complete(ctx context.Context, record Record, summary string) error {
	return s.finish(ctx, record, "succeeded", summary, StatusCompleted)
}

func (s *Service) Fail(ctx context.Context, record Record, summary string) error {
	return s.finish(ctx, record, "failed", summary, StatusFailed)
}

func (s *Service) Delete(ctx context.Context, repoRoot string, sessionID string) error {
	repoKey, err := RepoKey(repoRoot)
	if err != nil {
		return err
	}
	if err := ValidateSessionID(sessionID); err != nil {
		return err
	}
	return s.store.Delete(ctx, repoKey, sessionID)
}

func (s *Service) HistoryPath(repoRoot string, sessionID string) (string, error) {
	repoKey, err := RepoKey(repoRoot)
	if err != nil {
		return "", err
	}
	if err := ValidateSessionID(sessionID); err != nil {
		return "", err
	}
	return s.store.Path(repoKey, sessionID), nil
}

func (s *Service) finish(ctx context.Context, record Record, outcome string, summary string, status Status) error {
	now := s.now()
	runID := newRunID()
	record.Status = status
	record.UpdatedAt = now
	record.LastRunID = runID
	record.Runs = append(record.Runs, RunRecord{
		RunID:      runID,
		Prompt:     record.LastPrompt,
		Model:      record.Model,
		Outcome:    outcome,
		Summary:    summary,
		StartedAt:  record.UpdatedAt,
		FinishedAt: now,
	})
	record.CompactedHistory = CompactHistory(record.Runs, record.CompactedHistory, CompactionMessageSize)
	return s.store.Save(ctx, record)
}

func newRunID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "run-default"
	}
	return "run-" + hex.EncodeToString(b[:])
}
