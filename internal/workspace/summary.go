package workspace

import "fmt"

func SummarizeChanges(changes []Change) []string {
	lines := make([]string, 0, len(changes))
	for _, change := range changes {
		if change.Status == "blocked" {
			lines = append(lines, fmt.Sprintf("%s %s blocked: %s", change.Operation, change.Path, change.Reason))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s (%+d bytes)", change.Operation, change.Path, change.BytesChanged))
	}
	return lines
}
