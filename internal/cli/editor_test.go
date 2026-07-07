package cli

import "testing"

func TestEditorCommandPreference(t *testing.T) {
	t.Parallel()
	if got := editorCommand("code --wait", "vim"); got != "code --wait" {
		t.Fatalf("editorCommand() = %q", got)
	}
	if got := editorCommand("", "vim"); got != "vim" {
		t.Fatalf("editorCommand() = %q", got)
	}
	if got := editorCommand("", ""); got != "vi" {
		t.Fatalf("editorCommand() = %q", got)
	}
}

func TestStripPromptComments(t *testing.T) {
	t.Parallel()
	got := stripPromptComments("# helper\n\nBuild the feature\n  # ignored too\nwith tests\n")
	if got != "Build the feature\nwith tests" {
		t.Fatalf("stripPromptComments() = %q", got)
	}
}
