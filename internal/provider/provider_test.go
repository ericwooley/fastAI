package provider

import (
	"reflect"
	"testing"
)

func TestKnownReturnsDeterministicSupportedProviders(t *testing.T) {
	t.Parallel()
	known := Known()
	ids := make([]string, len(known))
	for i, info := range known {
		ids[i] = info.ID
	}
	want := []string{"deepseek", "github-copilot", "openai", "openrouter"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("Known IDs = %v, want %v", ids, want)
	}
}

func TestParseModel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		model        string
		wantProvider string
		wantModel    string
	}{
		{name: "plain model", model: "gpt-4.1", wantModel: "gpt-4.1"},
		{name: "known provider", model: "openrouter/deepseek/deepseek-chat", wantProvider: "openrouter", wantModel: "deepseek/deepseek-chat"},
		{name: "unknown provider-like prefix", model: "org/model", wantModel: "org/model"},
		{name: "trims whitespace", model: " openai/gpt-4.1 ", wantProvider: "openai", wantModel: "gpt-4.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotProvider, gotModel := ParseModel(tt.model)
			if gotProvider != tt.wantProvider || gotModel != tt.wantModel {
				t.Fatalf("ParseModel(%q) = (%q, %q), want (%q, %q)", tt.model, gotProvider, gotModel, tt.wantProvider, tt.wantModel)
			}
		})
	}
}

func TestLookupTrimsID(t *testing.T) {
	t.Parallel()
	info, err := Lookup(" openai ")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if info.ID != "openai" {
		t.Fatalf("info.ID = %q", info.ID)
	}
}
