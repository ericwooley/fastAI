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
