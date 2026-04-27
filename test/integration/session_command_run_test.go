package integration_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
	"github.com/ericwooley/fastAI/internal/commandexec"
	appsession "github.com/ericwooley/fastAI/internal/session"
	"github.com/ericwooley/fastAI/internal/workspace"
)

func TestResumedSessionCommandExecutionFlow(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	store := appsession.NewFileStore(filepath.Join(t.TempDir(), "sessions"))
	service := appsession.NewService(store, time.Now)
	record, resumed, err := service.Start(context.Background(), appsession.StartOptions{RepoRoot: repo, SessionID: "follow", Model: "github:gpt-4.1", Prompt: "first"})
	if err == nil || resumed || record.SessionID != "" {
		t.Fatalf("expected explicit missing session to fail, record=%+v resumed=%v err=%v", record, resumed, err)
	}
	record, _, err = service.Start(context.Background(), appsession.StartOptions{RepoRoot: repo, Model: "github:gpt-4.1", Prompt: "first"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	record.SessionID = "follow"
	if err := store.Save(context.Background(), record); err != nil {
		t.Fatalf("save follow: %v", err)
	}
	resumedRecord, resumed, err := service.Start(context.Background(), appsession.StartOptions{RepoRoot: repo, SessionID: "follow", Model: "github:gpt-4.1", Prompt: "run command printf ok"})
	if err != nil || !resumed {
		t.Fatalf("resume: record=%+v resumed=%v err=%v", resumedRecord, resumed, err)
	}
	runner := agent.NewLocalRunner(workspace.NewEditor(repo), commandexec.NewExecutor(repo), githubmodels.NewValidator(nil, "", "", githubmodels.WithOfflineValidation()))
	result, err := runner.Run(context.Background(), agent.Request{Prompt: resumedRecord.LastPrompt, Model: resumedRecord.Model, SessionID: resumedRecord.SessionID, AccessToken: "token"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.CommandResults) != 1 || result.CommandResults[0].ExitCode != 0 {
		t.Fatalf("unexpected command results: %+v", result.CommandResults)
	}
}
