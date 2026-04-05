package analysis_test

import (
	"reflect"
	"testing"
)

func TestWhitespaceTokenizer(t *testing.T) {
	tokenizer := NewWhitespaceTokenizer()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple sentence",
			input: "the quick brown fox",
			want:  []string{"the", "quick", "brown", "fox"},
		},
		{
			name:  "leading and trailing spaces",
			input: "  hello world  ",
			want:  []string{"hello", "world"},
		},
		{
			name:  "multiple internal spaces",
			input: "a  b   c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "tabs and newlines",
			input: "go\tis\ngreat",
			want:  []string{"go", "is", "great"},
		},
		{
			name:  "single word",
			input: "hello",
			want:  []string{"hello"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "only whitespace",
			input: "   \t\n  ",
			want:  []string{},
		},
		{
			name:  "punctuation attached to words",
			input: "hello, world!",
			want:  []string{"hello,", "world!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenizer.Tokenize(tt.input)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tokenize(%q)\n  got:  %v\n  want: %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLowercaseFilter(t *testing.T) {
	filter := NewLowercaseFilter()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "all uppercase",
			input: []string{"HELLO", "WORLD"},
			want:  []string{"hello", "world"},
		},
		{
			name:  "mixed case",
			input: []string{"Go", "Is", "GREAT"},
			want:  []string{"go", "is", "great"},
		},
		{
			name:  "already lowercase",
			input: []string{"already", "lower"},
			want:  []string{"already", "lower"},
		},
		{
			name:  "empty slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "unicode characters",
			input: []string{"ÜNÎCÖDÉ"},
			want:  []string{"ünîcödé"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filter.Filter(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter(%v)\n  got:  %v\n  want: %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPunctuationFilter(t *testing.T) {
	filter := NewPunctuationFilter()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "trailing comma and exclamation",
			input: []string{"hello,", "world!"},
			want:  []string{"hello", "world"},
		},
		{
			name:  "leading punctuation",
			input: []string{"(hello", ")world"},
			want:  []string{"hello", "world"},
		},
		{
			name:  "only punctuation becomes empty — discarded",
			input: []string{"...", "---", "hello"},
			want:  []string{"hello"},
		},
		{
			name:  "internal punctuation preserved",
			input: []string{"don't", "it's"},
			want:  []string{"don't", "it's"},
		},
		{
			name:  "clean tokens unchanged",
			input: []string{"go", "search"},
			want:  []string{"go", "search"},
		},
		{
			name:  "empty slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "period at end",
			input: []string{"end."},
			want:  []string{"end"},
		},
		{
			name:  "quoted word",
			input: []string{`"quoted"`},
			want:  []string{"quoted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filter.Filter(tt.input)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter(%v)\n  got:  %v\n  want: %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPipelineAnalyzer(t *testing.T) {
	newStandardAnalyzer := func() *PipelineAnalyzer {
		return NewPipelineAnalyzer(
			NewWhitespaceTokenizer(),
			NewLowercaseFilter(),
			NewPunctuationFilter(),
		)
	}

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "typical sentence",
			input: "The Quick Brown Fox!",
			want:  []string{"the", "quick", "brown", "fox"},
		},
		{
			name:  "punctuation and mixed case",
			input: "Hello, World.",
			want:  []string{"hello", "world"},
		},
		{
			name:  "query-style input",
			input: "What is Go?",
			want:  []string{"what", "is", "go"},
		},
		{
			name:  "already normalized",
			input: "go is great",
			want:  []string{"go", "is", "great"},
		},
		{
			name:  "empty input",
			input: "",
			want:  []string{},
		},
		{
			name:  "only punctuation",
			input: "!!! ???",
			want:  []string{},
		},
		{
			name:  "numbers preserved",
			input: "Go 1.22 released",
			want:  []string{"go", "1.22", "released"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := newStandardAnalyzer()
			got := analyzer.Analyze(tt.input)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Analyze(%q)\n  got:  %v\n  want: %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPipelineAnalyzer_NoFilters(t *testing.T) {
	analyzer := NewPipelineAnalyzer(NewWhitespaceTokenizer())
	got := analyzer.Analyze("Hello, World!")
	want := []string{"Hello,", "World!"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPipelineAnalyzer_FilterOrder(t *testing.T) {
	analyzer := NewPipelineAnalyzer(
		NewWhitespaceTokenizer(),
		NewPunctuationFilter(),
		NewLowercaseFilter(),
	)
	got := analyzer.Analyze("HELLO, WORLD.")
	want := []string{"hello", "world"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
