package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type OperationType string

const (
	OperationCreate OperationType = "create"
	OperationUpdate OperationType = "update"
	OperationPatch  OperationType = "patch"
	OperationDelete OperationType = "delete"
)

type Operation struct {
	Type    OperationType
	Path    string
	Content string
}

type PatchOperation struct {
	Path       string
	Old        string
	New        string
	ReplaceAll bool
}

type Change struct {
	Path         string `json:"path"`
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	BytesChanged int64  `json:"bytes_changed"`
}

type ReadResult struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Content string `json:"content,omitempty"`
	Bytes   int64  `json:"bytes"`
}

type Editor struct {
	repoRoot string
}

func NewEditor(repoRoot string) *Editor {
	return &Editor{repoRoot: repoRoot}
}

func (e *Editor) Apply(ctx context.Context, op Operation) (Change, error) {
	rel, abs, err := NormalizeRepoPath(e.repoRoot, op.Path)
	change := Change{Path: rel, Operation: string(op.Type)}
	if err != nil {
		change.Path = op.Path
		change.Status = "blocked"
		change.Reason = err.Error()
		return change, err
	}
	change.Status = "applied"
	switch op.Type {
	case OperationCreate, OperationUpdate:
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return change, err
		}
		oldSize := int64(0)
		if info, err := os.Stat(abs); err == nil {
			oldSize = info.Size()
		}
		if err := os.WriteFile(abs, []byte(op.Content), 0o644); err != nil {
			return change, err
		}
		change.BytesChanged = int64(len(op.Content)) - oldSize
		return change, nil
	case OperationPatch:
		return e.Patch(ctx, PatchOperation{Path: op.Path, Old: "", New: op.Content})
	case OperationDelete:
		info, err := os.Stat(abs)
		if err != nil {
			return change, err
		}
		if err := os.Remove(abs); err != nil {
			return change, err
		}
		change.BytesChanged = -info.Size()
		return change, nil
	default:
		return change, os.ErrInvalid
	}
}

func (e *Editor) Read(_ context.Context, path string) (ReadResult, error) {
	rel, abs, err := NormalizeRepoPath(e.repoRoot, path)
	result := ReadResult{Path: rel}
	if err != nil {
		result.Path = path
		result.Status = "blocked"
		result.Reason = err.Error()
		return result, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		result.Status = "failed"
		result.Reason = err.Error()
		return result, err
	}
	result.Status = "read"
	result.Content = string(data)
	result.Bytes = int64(len(data))
	return result, nil
}

func (e *Editor) Patch(_ context.Context, op PatchOperation) (Change, error) {
	rel, abs, err := NormalizeRepoPath(e.repoRoot, op.Path)
	change := Change{Path: rel, Operation: string(OperationPatch)}
	if err != nil {
		change.Path = op.Path
		change.Status = "blocked"
		change.Reason = err.Error()
		return change, err
	}
	if op.Old == "" {
		change.Status = "failed"
		change.Reason = "old text is required"
		return change, os.ErrInvalid
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		change.Status = "failed"
		change.Reason = err.Error()
		return change, err
	}
	content := string(data)
	count := strings.Count(content, op.Old)
	if count == 0 {
		change.Status = "failed"
		change.Reason = "old text not found"
		return change, os.ErrNotExist
	}
	updated := strings.Replace(content, op.Old, op.New, 1)
	if op.ReplaceAll {
		updated = strings.ReplaceAll(content, op.Old, op.New)
	}
	if err := os.WriteFile(abs, []byte(updated), 0o644); err != nil {
		change.Status = "failed"
		change.Reason = err.Error()
		return change, err
	}
	change.Status = "applied"
	change.BytesChanged = int64(len(updated) - len(content))
	return change, nil
}

func (e *Editor) ApplyAll(ctx context.Context, ops []Operation) ([]Change, error) {
	changes := make([]Change, 0, len(ops))
	for _, op := range ops {
		change, err := e.Apply(ctx, op)
		changes = append(changes, change)
		if err != nil {
			return changes, err
		}
	}
	return changes, nil
}
