package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnsafePath = errors.New("path escapes repository boundary")

func FindRepoRoot(start string) (string, error) {
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for {
		if _, err := os.Stat(filepath.Join(abs, ".git")); err == nil {
			return filepath.Clean(abs), nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("repository root not found from %s", start)
		}
		abs = parent
	}
}

func NormalizeRepoPath(repoRoot string, requested string) (string, string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", "", fmt.Errorf("repo root is required")
	}
	if strings.TrimSpace(requested) == "" {
		return "", "", fmt.Errorf("path is required")
	}
	rootAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", "", err
	}
	rootAbs = filepath.Clean(rootAbs)

	target := requested
	if !filepath.IsAbs(target) {
		target = filepath.Join(rootAbs, target)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", "", err
	}
	targetAbs = filepath.Clean(targetAbs)
	if err := ensureWithin(rootAbs, targetAbs); err != nil {
		return "", "", err
	}
	if err := ensureExistingParentWithin(rootAbs, targetAbs); err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", "", err
	}
	if rel == "." {
		return rel, targetAbs, nil
	}
	return filepath.ToSlash(rel), targetAbs, nil
}

func ensureExistingParentWithin(rootAbs string, targetAbs string) error {
	path := targetAbs
	if _, err := os.Stat(path); err != nil {
		path = filepath.Dir(path)
		for {
			if _, statErr := os.Stat(path); statErr == nil {
				break
			}
			parent := filepath.Dir(path)
			if parent == path {
				return ErrUnsafePath
			}
			path = parent
		}
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	return ensureWithin(rootAbs, realPath)
}

func ensureWithin(rootAbs string, targetAbs string) error {
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return ErrUnsafePath
	}
	return nil
}
