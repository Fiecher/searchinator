package analysis

import (
	"reflect"
	"testing"
)

func TestPorterStemmer_Stem(t *testing.T) {
	tests := []struct {
		word string
		want string
	}{

		{"caresses", "caress"},
		{"ponies", "poni"},
		{"cats", "cat"},

		{"agreed", "agre"},
		{"plastered", "plaster"},
		{"motoring", "motor"},
		{"sing", "sing"},

		{"hopping", "hop"},
		{"falling", "fall"},
		{"filing", "file"},

		{"relational", "relat"},
		{"conditional", "condit"},
		{"rational", "ration"},
		{"validate", "valid"},
		{"happiness", "happi"},
		{"hopeful", "hope"},
		{"goodness", "good"},
		{"revival", "reviv"},
		{"allowance", "allow"},
		{"adjustment", "adjust"},
		{"dependent", "depend"},
		{"adoption", "adopt"},

		{"probate", "probat"},
		{"controll", "control"},

		{"running", "run"},
		{"runs", "run"},
		{"programming", "program"},
		{"compiled", "compil"},
		{"functional", "function"},

		{"go", "go"},
		{"is", "is"},
	}

	s := NewPorterStemmer()
	for _, tt := range tests {
		got := s.Filter([]string{tt.word})[0]
		if got != tt.want {
			t.Errorf("stem(%q) = %q, want %q", tt.word, got, tt.want)
		}
	}
}

func TestPorterStemmer_ConflatesRelatedForms(t *testing.T) {
	s := NewPorterStemmer()
	forms := []string{"connect", "connected", "connecting", "connection", "connections"}
	out := s.Filter(forms)
	first := out[0]
	for i, stemmed := range out {
		if stemmed != first {
			t.Errorf("stem(%q) = %q, want all forms to share stem %q", forms[i], stemmed, first)
		}
	}
}

func TestPorterStemmer_PassesThroughNonASCII(t *testing.T) {
	s := NewPorterStemmer()
	tokens := []string{"телевизор", "go2", "naïve"}
	got := s.Filter(tokens)
	if !reflect.DeepEqual(got, tokens) {
		t.Errorf("non-ascii tokens should pass through unchanged: got %v, want %v", got, tokens)
	}
}
