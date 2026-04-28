package cli

import "testing"

func TestValidateRunInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   RunInput
		wantErr bool
	}{
		{name: "valid", input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot"}},
		{name: "valid provider", input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "openai"}},
		{name: "invalid provider", input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "anthropic"}, wantErr: true},
		{name: "missing prompt", input: RunInput{Model: "gpt-4.1", Provider: "github-copilot"}, wantErr: true},
		{name: "missing model", input: RunInput{Prompt: "do work"}, wantErr: true},
		{name: "missing provider", input: RunInput{Prompt: "do work", Model: "gpt-4.1"}, wantErr: true},
		{name: "valid session", input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", SessionID: "follow-up_1"}},
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
			name:  "trims model and provider",
			input: RunInput{Prompt: "do work", Model: " gpt-4.1 ", Provider: " github-copilot "},
			want:  RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot"},
		},
		{
			name:  "hashes explicit session",
			input: RunInput{Prompt: "do work", Model: "gpt-4.1", Provider: "github-copilot", SessionID: "follow"},
			want:  RunInput{Prompt: "do work", Model: "gpt-4.1", SessionID: "a4010945e4bd924bc2a890a2effea0e6", Provider: "github-copilot"},
		},
		{
			name:  "keeps slash model unchanged",
			input: RunInput{Prompt: "do work", Model: "deepseek/deepseek-chat", Provider: "openrouter"},
			want:  RunInput{Prompt: "do work", Model: "deepseek/deepseek-chat", Provider: "openrouter"},
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
