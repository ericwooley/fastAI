package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeRepoPath(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.Mkdir(filepath.Join(repo, "dir"), 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}
	tests := []struct {
		name    string
		path    string
		wantRel string
		wantErr bool
	}{
		{name: "relative", path: "dir/file.txt", wantRel: "dir/file.txt"},
		{name: "dot clean", path: "dir/../file.txt", wantRel: "file.txt"},
		{name: "traversal", path: "../outside.txt", wantErr: true},
		{name: "absolute outside", path: filepath.Join(filepath.Dir(repo), "outside.txt"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rel, _, err := NormalizeRepoPath(repo, tt.path)
			if tt.wantErr {
				if !errors.Is(err, ErrUnsafePath) {
					t.Fatalf("expected unsafe path, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rel != tt.wantRel {
				t.Fatalf("rel = %q, want %q", rel, tt.wantRel)
			}
		})
	}
}

func TestFindRepoRoot(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	nested := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	found, err := FindRepoRoot(nested)
	if err != nil {
		t.Fatalf("find repo: %v", err)
	}
	if found != repo {
		t.Fatalf("found %q, want %q", found, repo)
	}
}
