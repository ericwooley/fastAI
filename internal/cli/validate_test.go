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
		{name: "valid", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot"}},
		{name: "valid provider", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "openai"}},
		{name: "invalid provider", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "anthropic"}, wantErr: true},
		{name: "missing prompt", input: ResolvedRunInput{Model: "gpt-4.1", Provider: "github-copilot"}, wantErr: true},
		{name: "missing model", input: ResolvedRunInput{Prompt: "do work"}, wantErr: true},
		{name: "missing provider", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1"}, wantErr: true},
		{name: "valid session", input: ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", SessionID: "follow-up_1"}},
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
			input: RunInput{Prompt: "do work", Model: " gpt-4.1 ", Provider: " github-copilot "},
			want:  ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: agent.AllPermissions()},
		},
		{
			name:  "hashes explicit session",
			input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", SessionID: "follow"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", SessionID: "a4010945e4bd924bc2a890a2effea0e6", Provider: "github-copilot", Permissions: agent.AllPermissions()},
		},
		{
			name:  "keeps slash model unchanged",
			input: RunInput{Prompt: "do work", Model: "deepseek/deepseek-chat", Provider: "openrouter"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "deepseek/deepseek-chat", Provider: "openrouter", Permissions: agent.AllPermissions()},
		},
		{
			name:  "parses explicit permissions",
			input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: "read,write"},
			want:  ResolvedRunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", Permissions: agent.Permissions{Set: true, Read: true, Write: true}},
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
