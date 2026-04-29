package agent

import (
	"context"
	"reflect"
	"sort"
	"testing"
)

type capturePromptRunner struct {
	tools []string
}

func (r *capturePromptRunner) RunPrompt(_ context.Context, req PromptRunRequest) (PromptRunResult, error) {
	r.tools = make([]string, 0, len(req.Tools))
	for _, tool := range req.Tools {
		r.tools = append(r.tools, tool.Name())
	}
	sort.Strings(r.tools)
	return PromptRunResult{Text: "ok"}, nil
}

func TestLocalRunnerPermissionsSelectTools(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		permissions Permissions
		wantTools   []string
	}{
		{name: "default allows all", wantTools: []string{"delete_file", "patch_file", "read_file", "run_command", "write_file"}},
		{name: "read only", permissions: Permissions{Set: true, Read: true}, wantTools: []string{"read_file"}},
		{name: "write only", permissions: Permissions{Set: true, Write: true}, wantTools: []string{"delete_file", "patch_file", "write_file"}},
		{name: "execute only", permissions: Permissions{Set: true, Execute: true}, wantTools: []string{"run_command"}},
		{name: "read write", permissions: Permissions{Set: true, Read: true, Write: true}, wantTools: []string{"delete_file", "patch_file", "read_file", "write_file"}},
		{name: "none", permissions: Permissions{Set: true}, wantTools: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			promptRunner := &capturePromptRunner{}
			runner := NewLocalRunnerWithPromptRunner(nil, nil, nil, promptRunner)

			_, err := runner.Run(context.Background(), Request{Prompt: "do work", Model: "model", Permissions: tt.permissions})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if !reflect.DeepEqual(promptRunner.tools, tt.wantTools) {
				t.Fatalf("tools = %+v, want %+v", promptRunner.tools, tt.wantTools)
			}
		})
	}
}
