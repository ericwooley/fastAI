package cli

import "testing"

func TestValidateRunInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   RunInput
		wantErr bool
	}{
		{name: "valid", input: RunInput{Prompt: "do work", Model: "github:gpt-4.1"}},
		{name: "valid provider", input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "openai"}},
		{name: "invalid provider", input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "anthropic"}, wantErr: true},
		{name: "missing prompt", input: RunInput{Model: "github:gpt-4.1"}, wantErr: true},
		{name: "missing model", input: RunInput{Prompt: "do work"}, wantErr: true},
		{name: "invalid session", input: RunInput{Prompt: "do work", Model: "github:gpt-4.1", SessionID: "../bad"}, wantErr: true},
		{name: "valid session", input: RunInput{Prompt: "do work", Model: "github:gpt-4.1", SessionID: "follow-up_1"}},
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
		want    RunInput
		wantErr bool
	}{
		{
			name:  "defaults to GitHub Copilot",
			input: RunInput{Prompt: "do work", Model: "gpt-4.1"},
			want:  RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot"},
		},
		{
			name:  "uses provider prefix",
			input: RunInput{Prompt: "do work", Model: "openrouter/deepseek/deepseek-chat"},
			want:  RunInput{Prompt: "do work", Model: "deepseek/deepseek-chat", Provider: "openrouter"},
		},
		{
			name:  "keeps matching explicit provider",
			input: RunInput{Prompt: "do work", Model: "openai/gpt-4.1", Provider: "openai"},
			want:  RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "openai"},
		},
		{
			name:    "rejects conflicting provider",
			input:   RunInput{Prompt: "do work", Model: "openrouter/deepseek-chat", Provider: "openai"},
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
			if got != tt.want {
				t.Fatalf("ResolveRunInput() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
