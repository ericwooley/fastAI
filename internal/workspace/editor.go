package workspace

import (
	"context"
	"os"
	"path/filepath"
)

type OperationType string

const (
	OperationCreate OperationType = "create"
	OperationUpdate OperationType = "update"
	OperationDelete OperationType = "delete"
)

type Operation struct {
	Type    OperationType
	Path    string
	Content string
}

type Change struct {
	Path         string `json:"path"`
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	BytesChanged int64  `json:"bytes_changed"`
}

type Editor struct {
	repoRoot string
}

func NewEditor(repoRoot string) *Editor {
	return &Editor{repoRoot: repoRoot}
}

func (e *Editor) Apply(_ context.Context, op Operation) (Change, error) {
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
