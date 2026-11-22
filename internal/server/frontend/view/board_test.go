package view

import (
	"testing"

	"github.com/sj14/darts-o-mat/internal/common/dartboard"
	"github.com/sj14/darts-o-mat/internal/server/domain/throw"
)

// TestBoardFields checks that every field of the drawn board is a dart that can be entered,
// once.
func TestBoardFields(t *testing.T) {
	t.Parallel()
	if len(boardFields) != 82 {
		t.Fatalf("got %d fields, want 82 (20 numbers with 4 fields, bull and bullseye)", len(boardFields))
	}
	type field struct {
		number  uint8
		segment dartboard.Segment
	}
	seen := map[field]bool{}
	for _, f := range boardFields {
		key := field{f.Number, f.Segment}
		if seen[key] {
			t.Errorf("field %d %s appears twice", f.Number, f.Segment)
		}
		seen[key] = true
		if err := (throw.Dart{Number: f.Number, Segment: f.Segment}).Validate(); err != nil {
			t.Errorf("field %d %s isn't a valid dart: %v", f.Number, f.Segment, err)
		}
		if f.Path == "" {
			t.Errorf("field %d %s has no path", f.Number, f.Segment)
		}
	}
}

func TestFieldVals(t *testing.T) {
	t.Parallel()
	f := boardField{Number: 20, Segment: dartboard.Inner}
	if got, want := fieldVals(f), `{"number":"20","segment":"inner"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
