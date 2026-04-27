package workspace

import "testing"

func TestSummarizeChanges(t *testing.T) {
	t.Parallel()
	lines := SummarizeChanges([]Change{
		{Path: "a.txt", Operation: "update", Status: "applied", BytesChanged: 3},
		{Path: "../b.txt", Operation: "create", Status: "blocked", Reason: "unsafe"},
	})
	if len(lines) != 2 {
		t.Fatalf("len = %d", len(lines))
	}
	if lines[0] != "update a.txt (+3 bytes)" {
		t.Fatalf("line 0 = %q", lines[0])
	}
	if lines[1] != "create ../b.txt blocked: unsafe" {
		t.Fatalf("line 1 = %q", lines[1])
	}
}
