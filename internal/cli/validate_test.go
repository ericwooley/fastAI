package cli

import (
	"reflect"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent"
)

func TestValidateRunInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   ResolvedRunInput
		wantErr bool
	}{
		{name: "valid", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: agent.AllPermissions()}},
		{name: "valid provider", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "openai", Permissions: agent.AllPermissions()}},
		{name: "invalid provider", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "anthropic", Permissions: agent.AllPermissions()}, wantErr: true},
		{name: "missing prompt", input: ResolvedRunInput{Model: "gpt-4.1", Provider: "github-copilot", Permissions: agent.AllPermissions()}, wantErr: true},
		{name: "missing model", input: ResolvedRunInput{Prompt: "do work", Provider: "github-copilot", Permissions: agent.AllPermissions()}, wantErr: true},
		{name: "missing provider", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Permissions: agent.AllPermissions()}, wantErr: true},
		{name: "missing permissions", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot"}, wantErr: true},
		{name: "valid session", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", SessionID: "follow-up_1", Permissions: agent.AllPermissions()}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateRunInput(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveRunInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   RunInput
		want    ResolvedRunInput
		wantErr bool
	}{
		{
			name:  "trims model and provider",
			input: RunInput{Prompt: "do work", Model: " gpt-4.1 ", Provider: " github-copilot ", Permissions: "all"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: agent.AllPermissions()},
		},
		{
			name:  "hashes explicit session",
			input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", SessionID: "follow", Permissions: "all"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", SessionID: "a4010945e4bd924bc2a890a2effea0e6", Provider: "github-copilot", Permissions: agent.AllPermissions()},
		},
		{
			name:  "keeps slash model unchanged",
			input: RunInput{Prompt: "do work", Model: "deepseek/deepseek-chat", Provider: "openrouter", Permissions: "all"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "deepseek/deepseek-chat", Provider: "openrouter", Permissions: agent.AllPermissions()},
		},
		{
			name:  "leaves missing permissions unresolved",
			input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot"},
		},
		{
			name:  "parses explicit permissions",
			input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: "read,write"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: agent.Permissions{Set: true, Read: true, Write: true}},
		},
		{
			name:  "parses all permissions",
			input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: "all"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: agent.AllPermissions()},
		},
		{
			name:  "parses none permissions",
			input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: "none"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: agent.Permissions{Set: true}},
		},
		{
			name:    "rejects unknown permission",
			input:   RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: "read,admin"},
			wantErr: true,
		},
		{
			name:    "rejects none mixed with permission",
			input:   RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: "none,read"},
			wantErr: true,
		},
		{
			name:    "rejects all mixed with permission",
			input:   RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: "all,read"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ResolveRunInput(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ResolveRunInput() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveRunInputUsesEnvironmentDefaults(t *testing.T) {
	t.Setenv("FASTAI_DEFAULT_MODEL", " github:gpt-5-mini ")
	t.Setenv("FASTAI_DEFAULT_PROVIDER", " github-copilot ")
	t.Setenv("FASTAI_DEFAULT_PERMISSIONS", " all ")

	got, err := ResolveRunInput(RunInput{Prompt: "do work"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := ResolvedRunInput{Prompt: "do work", Model: "github:gpt-5-mini", Provider: "github-copilot", Permissions: agent.AllPermissions()}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveRunInput() = %+v, want %+v", got, want)
	}
}

func TestResolveRunInputFlagsOverrideEnvironmentDefaults(t *testing.T) {
	t.Setenv("FASTAI_DEFAULT_MODEL", "env-model")
	t.Setenv("FASTAI_DEFAULT_PROVIDER", "openai")
	t.Setenv("FASTAI_DEFAULT_PERMISSIONS", "all")

	got, err := ResolveRunInput(RunInput{Prompt: "do work", Model: "flag-model", Provider: "github-copilot", Permissions: "execute"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := ResolvedRunInput{Prompt: "do work", Model: "flag-model", Provider: "github-copilot", Permissions: agent.Permissions{Set: true, Execute: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveRunInput() = %+v, want %+v", got, want)
	}
}
