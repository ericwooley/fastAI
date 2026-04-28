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

func TestEditorReadAndPatch(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	editor := NewEditor(repo)
	ctx := context.Background()
	path := filepath.Join(repo, "dir", "file.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("hello world\nhello world\n"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	read, err := editor.Read(ctx, "dir/file.txt")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.Status != "read" || read.Path != "dir/file.txt" || read.Content != "hello world\nhello world\n" {
		t.Fatalf("unexpected read result: %+v", read)
	}

	change, err := editor.Patch(ctx, PatchOperation{Path: "dir/file.txt", Old: "hello world", New: "hello fastAI"})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if change.Operation != "patch" || change.Status != "applied" {
		t.Fatalf("unexpected patch change: %+v", change)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read patched file: %v", err)
	}
	if string(data) != "hello fastAI\nhello world\n" {
		t.Fatalf("patched content = %q", data)
	}

	change, err = editor.Patch(ctx, PatchOperation{Path: "dir/file.txt", Old: "hello", New: "hi"})
	if err != nil {
		t.Fatalf("patch first match: %v", err)
	}
	if change.Operation != "patch" || change.Status != "applied" {
		t.Fatalf("unexpected first-match patch change: %+v", change)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first-match patched file: %v", err)
	}
	if string(data) != "hi fastAI\nhello world\n" {
		t.Fatalf("first-match patched content = %q", data)
	}

	change, err = editor.Patch(ctx, PatchOperation{Path: "dir/file.txt", Old: "o", New: "O", ReplaceAll: true})
	if err != nil {
		t.Fatalf("patch replace all: %v", err)
	}
	if change.Operation != "patch" || change.Status != "applied" {
		t.Fatalf("unexpected replace-all patch change: %+v", change)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read replace-all patched file: %v", err)
	}
	if string(data) != "hi fastAI\nhellO wOrld\n" {
		t.Fatalf("replace-all patched content = %q", data)
	}

	read, err = editor.Read(ctx, "../blocked.txt")
	if !errors.Is(err, ErrUnsafePath) || read.Status != "blocked" {
		t.Fatalf("expected blocked read, got result=%+v err=%v", read, err)
	}

	_, err = editor.Patch(ctx, PatchOperation{Path: "../blocked.txt", Old: "x", New: "y"})
	if !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("expected blocked patch, got %v", err)
	}
}
