package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
	"github.com/ericwooley/fastAI/internal/auth"
	"github.com/ericwooley/fastAI/internal/commandexec"
	appsession "github.com/ericwooley/fastAI/internal/session"
	"github.com/ericwooley/fastAI/internal/workspace"
)

type Dependencies struct {
	Out            io.Writer
	Err            io.Writer
	AuthStore      auth.Store
	Authenticator  auth.Authenticator
	SessionService *appsession.Service
	Runner         agent.Runner
	RepoRoot       string
	Now            func() time.Time
}

func DefaultDependencies(out io.Writer, errw io.Writer) Dependencies {
	if out == nil {
		out = io.Discard
	}
	if errw == nil {
		errw = io.Discard
	}
	now := time.Now
	wd, _ := os.Getwd()
	repoRoot, repoErr := workspace.FindRepoRoot(wd)
	if repoErr != nil {
		repoRoot = wd
	}
	configDir, cfgErr := appsession.DefaultConfigDir()
	if cfgErr != nil {
		configDir = filepath.Join(repoRoot, "tmp", "fastAI-config")
	}
	var authenticator auth.Authenticator
	authenticator, authErr := auth.NewDeviceFlowAuthenticator()
	if authErr != nil {
		authenticator = auth.StaticAuthenticator{Err: authErr}
	}
	sessionStore := appsession.NewFileStore(filepath.Join(configDir, "sessions"))
	models := githubmodels.NewValidator(http.DefaultClient, "", "fastAI/0.1")
	return Dependencies{
		Out:            out,
		Err:            errw,
		AuthStore:      auth.NewFileStore(filepath.Join(configDir, "auth.json")),
		Authenticator:  authenticator,
		SessionService: appsession.NewService(sessionStore, now),
		Runner: agent.NewLocalRunnerWithPromptRunner(
			workspace.NewEditor(repoRoot),
			commandexec.NewExecutor(repoRoot),
			models,
			models,
		),
		RepoRoot: repoRoot,
		Now:      now,
	}
}

func Execute(ctx context.Context, args []string, deps Dependencies) int {
	deps = deps.withDefaults()
	cmd := NewRootCommand(deps)
	cmd.SetArgs(args)
	if err := cmd.ExecuteContext(ctx); err != nil {
		FormatError(deps.Err, err)
		return int(CodeForError(err))
	}
	return int(ExitSuccess)
}

func NewRootCommand(deps Dependencies) *cobra.Command {
	deps = deps.withDefaults()
	var model string
	var sessionID string
	cmd := &cobra.Command{
		Use:           "fastAI --model <model> [--session <identifier>] <prompt>",
		Short:         "Run an autonomous GitHub Copilot-backed coding agent",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := strings.Join(args, " ")
			if err := ValidateRunInput(RunInput{Prompt: prompt, Model: model, SessionID: sessionID}); err != nil {
				return err
			}
			repoRoot := deps.RepoRoot
			if repoRoot == "" {
				wd, _ := os.Getwd()
				found, err := workspace.FindRepoRoot(wd)
				if err != nil {
					return WrapError(ExitValidation, "repository root not found", "Run fastAI from inside a git repository.", err)
				}
				repoRoot = found
			}
			account, err := deps.AuthStore.Load(cmd.Context())
			if err != nil {
				return WrapRunError(err)
			}
			if err := account.Check(deps.Now()); err != nil {
				return WrapRunError(err)
			}
			record, _, err := deps.SessionService.Start(cmd.Context(), appsession.StartOptions{RepoRoot: repoRoot, SessionID: sessionID, Model: model, Prompt: prompt})
			if err != nil {
				return WrapRunError(err)
			}
			result, err := deps.Runner.Run(cmd.Context(), agent.Request{Prompt: prompt, Model: model, SessionID: record.SessionID, RepoRoot: repoRoot, AccessToken: account.AccessToken})
			if err != nil {
				_ = deps.SessionService.Fail(cmd.Context(), record, err.Error())
				return WrapRunError(err)
			}
			if result.SessionID == "" {
				result.SessionID = record.SessionID
			}
			if result.Model == "" {
				result.Model = model
			}
			if err := deps.SessionService.Complete(cmd.Context(), record, result.Summary); err != nil {
				return WrapRunError(err)
			}
			FormatRunSuccess(deps.Out, deps.Err, result)
			return nil
		},
	}
	cmd.SetOut(deps.Out)
	cmd.SetErr(deps.Err)
	cmd.Flags().StringVar(&model, "model", "", "model to use for the autonomous agent run")
	cmd.Flags().StringVar(&sessionID, "session", "", "session identifier for follow-up work")
	cmd.AddCommand(newLoginCommand(deps))
	return cmd
}

func (d Dependencies) withDefaults() Dependencies {
	if d.Out == nil {
		d.Out = io.Discard
	}
	if d.Err == nil {
		d.Err = io.Discard
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.AuthStore == nil || d.Authenticator == nil || d.SessionService == nil || d.Runner == nil {
		defaults := DefaultDependencies(d.Out, d.Err)
		if d.AuthStore == nil {
			d.AuthStore = defaults.AuthStore
		}
		if d.Authenticator == nil {
			d.Authenticator = defaults.Authenticator
		}
		if d.SessionService == nil {
			d.SessionService = defaults.SessionService
		}
		if d.Runner == nil {
			d.Runner = defaults.Runner
		}
		if d.RepoRoot == "" {
			d.RepoRoot = defaults.RepoRoot
		}
	}
	if d.RepoRoot == "" {
		panic(fmt.Sprintf("%s", "repo root dependency is empty"))
	}
	return d
}
