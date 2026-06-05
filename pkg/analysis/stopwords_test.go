package analysis

import (
	"reflect"
	"testing"
)

func TestStopWordsFilter(t *testing.T) {
	tests := []struct {
		name  string
		stop  []string
		input []string
		want  []string
	}{
		{
			name:  "removes listed stop words",
			stop:  []string{"the", "a", "is"},
			input: []string{"the", "quick", "fox", "is", "a", "animal"},
			want:  []string{"quick", "fox", "animal"},
		},
		{
			name:  "case insensitive matching",
			stop:  []string{"the"},
			input: []string{"The", "THE", "the", "cat"},
			want:  []string{"cat"},
		},
		{
			name:  "no stop words present",
			stop:  []string{"xyz"},
			input: []string{"go", "rust"},
			want:  []string{"go", "rust"},
		},
		{
			name:  "empty input",
			stop:  []string{"the"},
			input: []string{},
			want:  []string{},
		},
		{
			name:  "all tokens removed",
			stop:  []string{"a", "b"},
			input: []string{"a", "b", "a"},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewStopWordsFilter(tt.stop)
			got := f.Filter(tt.input)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultEnglishStopWords_FiltersCommonWords(t *testing.T) {
	f := NewStopWordsFilter(DefaultEnglishStopWords())
	got := f.Filter([]string{"the", "go", "is", "a", "language"})
	want := []string{"go", "language"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Filter() = %v, want %v", got, want)
	}
}
