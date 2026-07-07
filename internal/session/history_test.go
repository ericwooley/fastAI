package session

import (
	"strings"
	"testing"
	"time"
)

func TestCompactHistoryEveryTwentyMessages(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	var runs []RunRecord
	for i := 0; i < 10; i++ {
		runs = append(runs, RunRecord{
			RunID:      "run-test",
			Prompt:     "user asked for change",
			Outcome:    "succeeded",
			Summary:    "assistant made change",
			StartedAt:  base.Add(time.Duration(i) * time.Minute),
			FinishedAt: base.Add(time.Duration(i)*time.Minute + time.Second),
		})
	}

	compacted := CompactHistory(runs, nil, CompactionMessageSize)
	if len(compacted) != 1 {
		t.Fatalf("expected one compacted range, got %+v", compacted)
	}
	if compacted[0].MessageStart != 1 || compacted[0].MessageEnd != 20 {
		t.Fatalf("unexpected range: %+v", compacted[0])
	}
	if !strings.Contains(compacted[0].Summary, "2026-07-07T10:00:00Z") || !strings.Contains(compacted[0].Summary, "assistant made change") {
		t.Fatalf("summary missing grep-friendly details: %q", compacted[0].Summary)
	}

	compacted = CompactHistory(runs, compacted, CompactionMessageSize)
	if len(compacted) != 1 {
		t.Fatalf("expected idempotent compaction, got %+v", compacted)
	}

	for i := 10; i < 20; i++ {
		runs = append(runs, RunRecord{
			RunID:      "run-test",
			Prompt:     "next user change",
			Outcome:    "succeeded",
			Summary:    "next assistant change",
			StartedAt:  base.Add(time.Duration(i) * time.Minute),
			FinishedAt: base.Add(time.Duration(i)*time.Minute + time.Second),
		})
	}
	compacted = CompactHistory(runs, compacted, CompactionMessageSize)
	if len(compacted) != 2 || compacted[1].MessageStart != 21 || compacted[1].MessageEnd != 40 {
		t.Fatalf("expected second compacted range, got %+v", compacted)
	}
}

func TestBuildGlobalPromptIncludesCompactedRecentAndHistoryPath(t *testing.T) {
	t.Parallel()
	record := Record{
		CompactedHistory: []CompactedHistoryRecord{{
			MessageStart: 1,
			MessageEnd:   20,
			StartedAt:    time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC),
			FinishedAt:   time.Date(2026, 7, 7, 9, 30, 0, 0, time.UTC),
			Summary:      "worked on auth and tests",
		}},
		Runs: []RunRecord{
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{RunID: "run-old", Prompt: "already compacted", Summary: "old answer", Outcome: "succeeded"},
			{
				RunID:     "run-recent",
				Prompt:    "add global history",
				Outcome:   "succeeded",
				Summary:   "wired global session prompt",
				StartedAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	prompt := BuildGlobalPrompt(record, "continue", "/tmp/fastAI/global.json")
	for _, want := range []string{
		"Raw history file: /tmp/fastAI/global.json",
		"messages 1-20",
		"worked on auth and tests",
		"run-recent",
		"add global history",
		"Current request:\ncontinue",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "already compacted") {
		t.Fatalf("prompt should not expand compacted runs:\n%s", prompt)
	}
}
