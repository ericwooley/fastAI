package session

import (
	"fmt"
	"strings"
	"time"
)

const (
	GlobalSessionID       = "global"
	CompactionMessageSize = 20
)

func BuildGlobalPrompt(record Record, currentPrompt string, historyPath string) string {
	return BuildRememberedPrompt(record, currentPrompt, historyPath)
}

func BuildRememberedPrompt(record Record, currentPrompt string, historyPath string) string {
	currentPrompt = strings.TrimSpace(currentPrompt)
	history := rememberedPromptHistory(record, strings.TrimSpace(historyPath))
	if history == "" {
		return currentPrompt
	}
	return strings.TrimSpace(history + "\n\nCurrent request:\n" + currentPrompt)
}

func CompactHistory(runs []RunRecord, existing []CompactedHistoryRecord, everyMessages int) []CompactedHistoryRecord {
	if everyMessages <= 0 {
		return copyCompactedHistory(existing)
	}
	compacted := copyCompactedHistory(existing)
	nextMessage := nextCompactMessage(compacted)
	totalMessages := len(runs) * 2
	for totalMessages-nextMessage+1 >= everyMessages {
		endMessage := nextMessage + everyMessages - 1
		compacted = append(compacted, compactRuns(runs, nextMessage, endMessage))
		nextMessage = endMessage + 1
	}
	return compacted
}

func rememberedPromptHistory(record Record, historyPath string) string {
	if len(record.Runs) == 0 && len(record.CompactedHistory) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Persisted session context:\n")
	if historyPath != "" {
		b.WriteString("- Raw history file: ")
		b.WriteString(historyPath)
		b.WriteString("\n")
		b.WriteString("- To inspect older details, grep that file by timestamp, run id, or quoted text before relying on memory.\n")
	}
	if len(record.CompactedHistory) > 0 {
		b.WriteString("\nCompacted history:\n")
		for _, item := range record.CompactedHistory {
			b.WriteString(fmt.Sprintf("- messages %d-%d, %s to %s: %s\n", item.MessageStart, item.MessageEnd, formatTime(item.StartedAt), formatTime(item.FinishedAt), singleLine(item.Summary, 320)))
		}
	}
	recent := recentRuns(record)
	if len(recent) > 0 {
		b.WriteString("\nRecent conversation:\n")
		for _, run := range recent {
			b.WriteString(fmt.Sprintf("- %s %s %s\n", formatTime(run.StartedAt), run.RunID, run.Outcome))
			b.WriteString(fmt.Sprintf("  user: %s\n", singleLine(run.Prompt, 500)))
			if strings.TrimSpace(run.Summary) != "" {
				b.WriteString(fmt.Sprintf("  assistant: %s\n", singleLine(run.Summary, 500)))
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func recentRuns(record Record) []RunRecord {
	if len(record.Runs) == 0 {
		return nil
	}
	startMessage := nextCompactMessage(record.CompactedHistory)
	startRun := (startMessage - 1) / 2
	if startRun < 0 {
		startRun = 0
	}
	if startRun >= len(record.Runs) {
		return nil
	}
	return record.Runs[startRun:]
}

func compactRuns(runs []RunRecord, startMessage int, endMessage int) CompactedHistoryRecord {
	startRun := (startMessage - 1) / 2
	endRun := (endMessage - 1) / 2
	if startRun < 0 {
		startRun = 0
	}
	if endRun >= len(runs) {
		endRun = len(runs) - 1
	}
	if len(runs) == 0 || startRun > endRun {
		return CompactedHistoryRecord{MessageStart: startMessage, MessageEnd: endMessage}
	}
	selected := runs[startRun : endRun+1]
	var parts []string
	for _, run := range selected {
		user := singleLine(run.Prompt, 160)
		assistant := singleLine(run.Summary, 160)
		parts = append(parts, fmt.Sprintf("%s %s %s user=%q assistant=%q", formatTime(run.StartedAt), run.RunID, run.Outcome, user, assistant))
	}
	return CompactedHistoryRecord{
		MessageStart: startMessage,
		MessageEnd:   endMessage,
		StartedAt:    selected[0].StartedAt,
		FinishedAt:   selected[len(selected)-1].FinishedAt,
		Summary:      strings.Join(parts, " | "),
	}
}

func nextCompactMessage(compacted []CompactedHistoryRecord) int {
	next := 1
	for _, item := range compacted {
		if item.MessageEnd >= next {
			next = item.MessageEnd + 1
		}
	}
	return next
}

func copyCompactedHistory(existing []CompactedHistoryRecord) []CompactedHistoryRecord {
	if len(existing) == 0 {
		return nil
	}
	copied := make([]CompactedHistoryRecord, len(existing))
	copy(copied, existing)
	return copied
}

func singleLine(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "unknown-time"
	}
	return value.UTC().Format(time.RFC3339)
}
