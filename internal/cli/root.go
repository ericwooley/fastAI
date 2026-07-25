package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
	"github.com/ericwooley/fastAI/internal/auth"
	"github.com/ericwooley/fastAI/internal/commandexec"
	"github.com/ericwooley/fastAI/internal/provider"
	appsession "github.com/ericwooley/fastAI/internal/session"
	"github.com/ericwooley/fastAI/internal/workspace"
)

type Dependencies struct {
	Out              io.Writer
	Err              io.Writer
	In               io.Reader
	Version          string
	AuthStore        auth.Store
	Authenticator    auth.Authenticator
	SessionService   *appsession.Service
	Runner           agent.Runner
	RepoRoot         string
	WorkingDirectory string
	Now              func() time.Time
	Editor           PromptEditor
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
		In:             os.Stdin,
		Version:        "dev",
		AuthStore:      auth.NewFileStore(filepath.Join(configDir, "auth.json")),
		Authenticator:  authenticator,
		SessionService: appsession.NewService(sessionStore, now),
		Runner: agent.NewLocalRunnerWithPromptRunner(
			workspace.NewEditor(repoRoot),
			commandexec.NewExecutor(repoRoot),
			models,
			models,
		),
		RepoRoot:         repoRoot,
		WorkingDirectory: wd,
		Now:              now,
		Editor:           NewPromptEditor(os.Stdin, out, errw),
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
	var providerName string
	var permissions string
	var verbose bool
	var noSession bool
	var globalSession bool
	var newGlobalSession bool
	var history string
	cmd := &cobra.Command{
		Use:           "fastAI --provider <provider> --model <model> [--session <identifier>] [--globalSession] [--history [count]] <prompt>",
		Short:         "Run an autonomous coding agent",
		Version:       deps.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := strings.Join(args, " ")
			usesGlobalSession := globalSession || newGlobalSession
			if cmd.Flags().Changed("history") {
				return showSessionHistory(cmd, deps, sessionID, noSession, globalSession, newGlobalSession, history, args)
			}
			if noSession && strings.TrimSpace(sessionID) != "" {
				return NewError(ExitValidation, "--no-session cannot be used with --session", "Remove one of the session flags and retry.")
			}
			if noSession && usesGlobalSession {
				return NewError(ExitValidation, "--no-session cannot be used with --globalSession or --newGlobalSession", "Remove one of the session flags and retry.")
			}
			if strings.TrimSpace(sessionID) != "" && usesGlobalSession {
				return NewError(ExitValidation, "--session cannot be used with --globalSession or --newGlobalSession", "Choose a named session or the global session.")
			}
			input, err := ResolveRunInput(RunInput{Prompt: prompt, Model: model, SessionID: sessionID, Provider: providerName, Permissions: permissions})
			if err != nil {
				return err
			}
			if strings.TrimSpace(input.Prompt) == "" {
				inputWithoutPromptValidation := input
				inputWithoutPromptValidation.Prompt = "editor prompt placeholder"
				if err := ValidateRunInput(inputWithoutPromptValidation); err != nil {
					return err
				}
				editedPrompt, err := deps.Editor(cmd.Context())
				if err != nil {
					return WrapError(ExitAgent, "editor prompt failed", "Set VISUAL or EDITOR to a working editor, then retry.", err)
				}
				input.Prompt = editedPrompt
			}
			if err := ValidateRunInput(input); err != nil {
				return err
			}
			model = input.Model
			sessionID = input.SessionID

			repoRoot := deps.RepoRoot
			if repoRoot == "" {
				wd, _ := os.Getwd()
				found, err := workspace.FindRepoRoot(wd)
				if err != nil {
					return WrapError(ExitValidation, "repository root not found", "Run fastAI from inside a git repository.", err)
				}
				repoRoot = found
			}

			// Resolve access token based on provider
			accessToken, accountErr := resolveAccessToken(cmd.Context(), deps, input.Provider)
			if accountErr != nil {
				return WrapRunError(accountErr)
			}

			var runSessionID string
			var record appsession.Record
			runPrompt := input.Prompt
			if noSession {
				runSessionID = appsession.GenerateSessionID()
			} else {
				startSessionID := sessionID
				if usesGlobalSession {
					startSessionID = appsession.GlobalSessionID
					if newGlobalSession {
						if err := deps.SessionService.Delete(cmd.Context(), repoRoot, appsession.GlobalSessionID); err != nil {
							return WrapRunError(err)
						}
					}
				}
				var err error
				record, _, err = deps.SessionService.Start(cmd.Context(), appsession.StartOptions{RepoRoot: repoRoot, SessionID: startSessionID, Model: model, Prompt: input.Prompt})
				if err != nil {
					return WrapRunError(err)
				}
				runSessionID = record.SessionID
				historyPath, err := deps.SessionService.HistoryPath(repoRoot, runSessionID)
				if err != nil {
					return WrapRunError(err)
				}
				runPrompt = appsession.BuildRememberedPrompt(record, input.Prompt, historyPath)
			}

			// Create per-request prompt runner for non-default providers
			var runPromptRunner agent.PromptRunner
			if input.Provider != "github-copilot" {
				info, _ := provider.Lookup(input.Provider)
				providerAPIKey := accessToken
				if info.EnvKey != "" {
					if envVal := os.Getenv(info.EnvKey); envVal != "" {
						providerAPIKey = envVal
					}
				}
				var err error
				runPromptRunner, err = provider.NewPromptRunner(http.DefaultClient, input.Provider, providerAPIKey)
				if err != nil {
					return WrapRunError(err)
				}
			}

			result, err := deps.Runner.Run(cmd.Context(), agent.Request{
				Prompt:           runPrompt,
				Model:            model,
				SessionID:        runSessionID,
				RepoRoot:         repoRoot,
				WorkingDirectory: deps.WorkingDirectory,
				AccessToken:      accessToken,
				Provider:         input.Provider,
				Permissions:      input.Permissions,
				PromptRunner:     runPromptRunner,
				Progress:         newTelemetryProgress(deps.Err, verbose),
			})
			if err != nil {
				if !noSession {
					_ = deps.SessionService.Fail(cmd.Context(), record, err.Error())
				}
				return WrapRunError(err)
			}
			if noSession {
				result.SessionID = ""
			} else if result.SessionID == "" {
				result.SessionID = record.SessionID
			}
			if result.Model == "" {
				result.Model = model
			}
			if result.Provider == "" {
				result.Provider = input.Provider
			}
			if !noSession {
				if err := deps.SessionService.Complete(cmd.Context(), record, result.Summary); err != nil {
					return WrapRunError(err)
				}
			}
			FormatRunSuccess(deps.Out, deps.Err, result)
			return nil
		},
	}
	cmd.SetOut(deps.Out)
	cmd.SetErr(deps.Err)
	cmd.Flags().StringVar(&model, "model", "", "model to use for the autonomous agent run")
	cmd.Flags().StringVar(&providerName, "provider", "", "AI provider to use (e.g., openai, openrouter, deepseek, github-copilot)")
	cmd.Flags().StringVar(&sessionID, "session", "", "session identifier for follow-up work")
	cmd.Flags().StringVar(&permissions, "permissions", "", "comma-separated tool permissions: read, write, execute, or none")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show request timing and token usage on stderr")
	cmd.Flags().BoolVar(&noSession, "no-session", false, "run without saving session history")
	cmd.Flags().BoolVar(&globalSession, "globalSession", false, "continue the repository global session history")
	cmd.Flags().BoolVar(&newGlobalSession, "newGlobalSession", false, "wipe and start the repository global session history")
	cmd.Flags().StringVar(&history, "history", "", "show the selected session history, defaulting to the last 5 conversations")
	cmd.Flags().Lookup("history").NoOptDefVal = "5"
	cmd.AddCommand(newLoginCommand(deps))
	return cmd
}

func showSessionHistory(cmd *cobra.Command, deps Dependencies, sessionID string, noSession bool, globalSession bool, newGlobalSession bool, history string, args []string) error {
	if noSession {
		return NewError(ExitValidation, "--no-session cannot be used with --history", "Choose a saved session to inspect.")
	}
	if newGlobalSession {
		return NewError(ExitValidation, "--newGlobalSession cannot be used with --history", "Use --history to inspect history or --newGlobalSession to reset it.")
	}
	if strings.TrimSpace(sessionID) != "" && globalSession {
		return NewError(ExitValidation, "--session cannot be used with --globalSession", "Choose a named session or the global session.")
	}

	limit, err := resolveHistoryLimit(history, args)
	if err != nil {
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

	selectedSessionID := appsession.GlobalSessionID
	if strings.TrimSpace(sessionID) != "" {
		selectedSessionID = appsession.HashSessionID(sessionID)
	}
	record, err := deps.SessionService.Load(cmd.Context(), repoRoot, selectedSessionID)
	if err != nil {
		return WrapRunError(err)
	}
	if deps.Out != nil {
		fmt.Fprint(deps.Out, appsession.FormatConversationHistory(record, limit))
	}
	return nil
}

func resolveHistoryLimit(value string, args []string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "5"
	}
	if len(args) > 0 {
		if len(args) == 1 && value == "5" {
			if parsed, err := strconv.Atoi(strings.TrimSpace(args[0])); err == nil {
				return validateHistoryLimit(parsed)
			}
		}
		return 0, NewError(ExitValidation, "--history cannot be used with a prompt", "Use `fastAI --history` or `fastAI --history <number>`.")
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, NewError(ExitValidation, "--history must be a positive number", "Use `fastAI --history` or `fastAI --history 10`.")
	}
	return validateHistoryLimit(parsed)
}

func validateHistoryLimit(limit int) (int, error) {
	if limit <= 0 {
		return 0, NewError(ExitValidation, "--history must be a positive number", "Use `fastAI --history` or `fastAI --history 10`.")
	}
	return limit, nil
}

func resolveAccessToken(ctx context.Context, deps Dependencies, providerID string) (string, error) {
	if providerID == "github-copilot" {
		account, err := deps.AuthStore.Load(ctx)
		if err != nil {
			return "", err
		}
		if err := account.CheckForClient(deps.Now(), auth.CopilotClientID); err != nil {
			return "", err
		}
		return account.AccessToken, nil
	}
	info, _ := provider.Lookup(providerID)
	if info.EnvKey != "" {
		if apiKey := os.Getenv(info.EnvKey); apiKey != "" {
			return apiKey, nil
		}
	}
	return "", fmt.Errorf("no API key found for provider %q: set the %s environment variable", providerID, info.EnvKey)
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
	if d.In == nil {
		d.In = os.Stdin
	}
	if d.Version == "" {
		d.Version = "dev"
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
		if d.Editor == nil {
			d.Editor = defaults.Editor
		}
	}
	if d.Editor == nil {
		d.Editor = NewPromptEditor(d.In, d.Out, d.Err)
	}
	if d.RepoRoot == "" {
		panic(fmt.Sprintf("%s", "repo root dependency is empty"))
	}
	if d.WorkingDirectory == "" {
		d.WorkingDirectory = d.RepoRoot
	}
	return d
}
