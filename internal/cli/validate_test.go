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
