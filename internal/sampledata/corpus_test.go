package sampledata

import (
	"strings"
	"testing"
)

func TestCorpus_LoadsFromJSON(t *testing.T) {
	docs := Corpus()
	if len(docs) != 8 {
		t.Fatalf("Corpus() len = %d, want 8", len(docs))
	}
	for _, d := range docs {
		if d.ID == "" || d.Text == "" {
			t.Errorf("doc %+v missing ID or Text", d)
		}
		if _, ok := d.Meta["year"].(int); !ok {
			t.Errorf("doc %q year meta is not int: %T", d.ID, d.Meta["year"])
		}
	}
}

func TestExampleQuery_DataDriven(t *testing.T) {
	q := ExampleQuery()
	if q == "" || q == "type a query..." {
		t.Fatalf("ExampleQuery() = %q, want a derived example", q)
	}

	if !strings.Contains(q, "программирования") {
		t.Errorf("ExampleQuery() = %q, want it to include 'программирования'", q)
	}

	if q2 := ExampleQuery(); q2 != q {
		t.Errorf("ExampleQuery not deterministic: %q vs %q", q, q2)
	}
}
