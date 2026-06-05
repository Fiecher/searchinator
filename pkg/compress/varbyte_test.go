package compress

import (
	"reflect"
	"testing"
)

func TestEncodeDecodeGaps_RoundTrip(t *testing.T) {
	cases := [][]int{
		{},
		{0},
		{0, 1, 2, 3},
		{5, 200, 70000, 70001},
		{1, 128, 129, 16383, 16384, 2097152},
	}
	for _, in := range cases {
		got := DecodeGaps(EncodeGaps(in))
		if len(in) == 0 && len(got) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, in) {
			t.Errorf("round trip %v -> %v", in, got)
		}
	}
}

func TestEncodeGaps_ShrinksLargeRuns(t *testing.T) {
	positions := make([]int, 1000)
	for i := range positions {
		positions[i] = i
	}
	encoded := EncodeGaps(positions)
	raw := len(positions) * 8
	if len(encoded) >= raw {
		t.Errorf("expected compression below %d bytes, got %d", raw, len(encoded))
	}
}

func TestAppendUvarint_SingleByteForSmall(t *testing.T) {
	for v := uint64(0); v < 128; v++ {
		if got := AppendUvarint(nil, v); len(got) != 1 {
			t.Errorf("value %d should encode to 1 byte, got %d", v, len(got))
		}
	}
	if got := AppendUvarint(nil, 128); len(got) != 2 {
		t.Errorf("128 should need 2 bytes, got %d", len(got))
	}
}
