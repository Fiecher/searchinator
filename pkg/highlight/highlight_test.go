package highlight

import (
	"strings"
	"testing"
)

func lowerAnalyze(w string) []string {
	return []string{strings.Trim(strings.ToLower(w), ".,")}
}

func TestSnippet_MarksMatches(t *testing.T) {
	text := "Rust achieves memory safety without a garbage collector"
	got := Snippet(text, []string{"memory", "safety"}, lowerAnalyze, Options{Radius: 2, Open: "[", Close: "]"})
	if !strings.Contains(got, "[memory]") || !strings.Contains(got, "[safety]") {
		t.Errorf("expected matches marked, got %q", got)
	}
}

func TestSnippet_AddsEllipsisWhenTrimmed(t *testing.T) {
	text := "one two three four five six seven eight nine ten eleven twelve"
	got := Snippet(text, []string{"eight"}, lowerAnalyze, Options{Radius: 1, Open: "<", Close: ">"})
	if !strings.HasPrefix(got, "...") || !strings.HasSuffix(got, "...") {
		t.Errorf("expected ellipsis on both ends, got %q", got)
	}
	if !strings.Contains(got, "<eight>") {
		t.Errorf("expected <eight> highlighted, got %q", got)
	}
}

func TestSnippet_NoMatchReturnsPreview(t *testing.T) {
	text := "alpha beta gamma delta epsilon zeta eta theta"
	got := Snippet(text, []string{"missing"}, lowerAnalyze, Options{Radius: 2, Open: "*", Close: "*"})
	if strings.Contains(got, "*") {
		t.Errorf("no match should have no markers, got %q", got)
	}
	if !strings.HasPrefix(got, "alpha") {
		t.Errorf("preview should start at beginning, got %q", got)
	}
}

func TestSnippet_EmptyText(t *testing.T) {
	if got := Snippet("", []string{"x"}, lowerAnalyze, DefaultOptions()); got != "" {
		t.Errorf("empty text should give empty snippet, got %q", got)
	}
}

func TestHighlightSpans_MarksEveryOccurrence(t *testing.T) {
	text := "go is fast go is simple and go is reliable for big systems work"
	spans := HighlightSpans(text, []string{"go"}, lowerAnalyze, Options{Radius: 1, MaxFragments: 12})

	matches := 0
	for _, s := range spans {
		if s.Match {
			if s.Text != "go" {
				t.Errorf("match span = %q, want go", s.Text)
			}
			matches++
		}
	}
	if matches != 3 {
		t.Errorf("got %d match spans, want 3 (one per occurrence)", matches)
	}
}

func TestHighlightSpans_MergesAdjacentWindows(t *testing.T) {

	text := "alpha memory beta safety gamma"
	spans := HighlightSpans(text, []string{"memory", "safety"}, lowerAnalyze, Options{Radius: 2})
	for _, s := range spans {
		if !s.Match && containsEllipsis(s.Text) {
			t.Errorf("expected merged single fragment, found ellipsis in %q", s.Text)
		}
	}
}

func TestHighlights_String(t *testing.T) {
	text := "one two memory three four five six memory seven"
	got := Highlights(text, []string{"memory"}, lowerAnalyze, Options{Radius: 1, Open: "[", Close: "]", MaxFragments: 12})
	if want := "[memory]"; !strings.Contains(got, want) {
		t.Errorf("Highlights = %q, want it to contain %q", got, want)
	}
	if strings.Count(got, "[memory]") != 2 {
		t.Errorf("Highlights = %q, want two highlighted occurrences", got)
	}
	if !strings.Contains(got, "...") {
		t.Errorf("Highlights = %q, want an ellipsis between fragments", got)
	}
}

func containsEllipsis(s string) bool { return strings.Contains(s, "...") }
