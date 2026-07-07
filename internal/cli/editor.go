package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type PromptEditor func(context.Context) (string, error)

func NewPromptEditor(stdin io.Reader, stdout io.Writer, stderr io.Writer) PromptEditor {
	return func(ctx context.Context) (string, error) {
		return EditPrompt(ctx, EditorOptions{
			Editor: editorCommand(os.Getenv("VISUAL"), os.Getenv("EDITOR")),
			Stdin:  stdin,
			Stdout: stdout,
			Stderr: stderr,
		})
	}
}

type EditorOptions struct {
	Editor string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func EditPrompt(ctx context.Context, opts EditorOptions) (string, error) {
	editor := strings.TrimSpace(opts.Editor)
	if editor == "" {
		editor = "vi"
	}
	file, err := os.CreateTemp("", "fastAI-prompt-*.md")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if _, err := file.WriteString("# Write your fastAI prompt below. Lines starting with # are ignored.\n\n"); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	defer os.Remove(path)

	cmd := editorExec(ctx, editor, path)
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("open editor: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return stripPromptComments(string(data)), nil
}

func editorCommand(visual string, editor string) string {
	if strings.TrimSpace(visual) != "" {
		return visual
	}
	if strings.TrimSpace(editor) != "" {
		return editor
	}
	return "vi"
}

func editorExec(ctx context.Context, editor string, path string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", editor+" %1", "fastAI-editor", path)
	}
	return exec.CommandContext(ctx, "sh", "-c", editor+" \"$1\"", "fastAI-editor", path)
}

func stripPromptComments(value string) string {
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
