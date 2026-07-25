package agent

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

const agentInstructionsFilename = "AGENTS.md"

type agentInstruction struct {
	Path    string
	Content string
}

type readInstructionFile func(string) ([]byte, error)

func agentInstructionPaths(repoRoot string, workingDirectory string) ([]string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return nil, fmt.Errorf("repository root is required")
	}
	if strings.TrimSpace(workingDirectory) == "" {
		workingDirectory = repoRoot
	}

	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	workingDirectory, err = filepath.Abs(workingDirectory)
	if err != nil {
		return nil, err
	}

	relativeDirectory, err := filepath.Rel(root, workingDirectory)
	if err != nil {
		return nil, err
	}
	if relativeDirectory == ".." || strings.HasPrefix(relativeDirectory, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("working directory %q is outside repository root %q", workingDirectory, root)
	}

	paths := []string{filepath.Join(root, agentInstructionsFilename)}
	if relativeDirectory == "." {
		return paths, nil
	}

	currentDirectory := root
	for _, part := range strings.Split(relativeDirectory, string(filepath.Separator)) {
		currentDirectory = filepath.Join(currentDirectory, part)
		paths = append(paths, filepath.Join(currentDirectory, agentInstructionsFilename))
	}
	return paths, nil
}

func loadAgentInstructions(repoRoot string, workingDirectory string, readFile readInstructionFile) ([]agentInstruction, error) {
	paths, err := agentInstructionPaths(repoRoot, workingDirectory)
	if err != nil {
		return nil, err
	}

	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	instructions := make([]agentInstruction, 0, len(paths))
	for _, path := range paths {
		content, err := readFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		trimmedContent := strings.TrimSpace(string(content))
		if trimmedContent == "" {
			continue
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, agentInstruction{
			Path:    filepath.ToSlash(relativePath),
			Content: trimmedContent,
		})
	}
	return instructions, nil
}

func appendAgentInstructions(baseInstruction string, instructions []agentInstruction) string {
	if len(instructions) == 0 {
		return strings.TrimSpace(baseInstruction)
	}

	var result strings.Builder
	result.WriteString(strings.TrimSpace(baseInstruction))
	result.WriteString(`

# Repository instructions from AGENTS.md

The following AGENTS.md files are ordered from the repository root toward the current working directory. When instructions conflict, a later file takes priority because it is closer to the current working directory.`)

	for _, instruction := range instructions {
		result.WriteString("\n\n## ")
		result.WriteString(instruction.Path)
		result.WriteString("\n\n")
		result.WriteString(instruction.Content)
	}
	return strings.TrimSpace(result.String())
}
