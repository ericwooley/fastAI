package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEditorCreateUpdateDeleteAndBlock(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	editor := NewEditor(repo)
	ctx := context.Background()

	change, err := editor.Apply(ctx, Operation{Type: OperationCreate, Path: "dir/file.txt", Content: "hello"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if change.Path != "dir/file.txt" || change.Operation != "create" || change.Status != "applied" {
		t.Fatalf("unexpected create change: %+v", change)
	}

	change, err = editor.Apply(ctx, Operation{Type: OperationUpdate, Path: "dir/file.txt", Content: "hello world"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if change.BytesChanged != 6 {
		t.Fatalf("bytes changed = %d, want 6", change.BytesChanged)
	}

	change, err = editor.Apply(ctx, Operation{Type: OperationDelete, Path: "dir/file.txt"})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(repo, "dir/file.txt")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected deleted file, got %v", statErr)
	}
	if change.BytesChanged >= 0 {
		t.Fatalf("delete bytes should be negative: %+v", change)
	}

	change, err = editor.Apply(ctx, Operation{Type: OperationCreate, Path: "../blocked.txt", Content: "no"})
	if !errors.Is(err, ErrUnsafePath) || change.Status != "blocked" {
		t.Fatalf("expected blocked unsafe path, got change=%+v err=%v", change, err)
	}
}
