package provider

import (
	"testing"

	"google.golang.org/genai"
)

func TestMessagesFromContentsUsesConfiguredReasoningField(t *testing.T) {
	t.Parallel()
	contents := []*genai.Content{genai.NewContentFromParts([]*genai.Part{
		{Text: "private chain of thought", Thought: true},
		genai.NewPartFromText("visible answer"),
	}, genai.RoleModel)}

	messages := messagesFromContents(nil, contents, "reasoning_content")
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	if messages[0].ReasoningContent != "private chain of thought" {
		t.Fatalf("ReasoningContent = %q", messages[0].ReasoningContent)
	}
	if messages[0].ReasoningText != "" {
		t.Fatalf("ReasoningText = %q, want empty", messages[0].ReasoningText)
	}
	if messages[0].Content != "visible answer" {
		t.Fatalf("Content = %#v", messages[0].Content)
	}
}

func TestMessagesFromContentsDefaultsToReasoningText(t *testing.T) {
	t.Parallel()
	contents := []*genai.Content{genai.NewContentFromParts([]*genai.Part{{Text: "private chain of thought", Thought: true}}, genai.RoleModel)}

	messages := messagesFromContents(nil, contents, "")
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	if messages[0].ReasoningText != "private chain of thought" {
		t.Fatalf("ReasoningText = %q", messages[0].ReasoningText)
	}
	if messages[0].ReasoningContent != "" {
		t.Fatalf("ReasoningContent = %q, want empty", messages[0].ReasoningContent)
	}
}
