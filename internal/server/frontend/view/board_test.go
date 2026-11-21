package view

import (
	"testing"

	"github.com/sj14/darts-o-mat/internal/common/dartboard"
	"github.com/sj14/darts-o-mat/internal/common/i18n"
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
	if len(boardShapes) != 80 {
		t.Errorf("got %d ring fields, want 80 (the heatmap draws these)", len(boardShapes))
	}
}

func TestFieldVals(t *testing.T) {
	t.Parallel()
	if got, want := fieldVals(20, dartboard.Inner), `{"number": "20", "segment": "inner"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if got, want := fieldID(25, dartboard.Bullseye), "field-25-bullseye"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestFieldLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		lang    i18n.Lang
		number  uint8
		segment dartboard.Segment
		want    string
	}{
		{i18n.English, 20, dartboard.Inner, "Inner single 20"},
		{i18n.English, 20, dartboard.Outer, "Outer single 20"},
		{i18n.English, 20, dartboard.Double, "Double 20"},
		{i18n.English, 20, dartboard.Triple, "Triple 20"},
		{i18n.English, 25, dartboard.Bull, "Bull"},
		{i18n.English, 25, dartboard.Bullseye, "Bullseye"},
		{i18n.German, 7, dartboard.Inner, "Inneres Single 7"},
		{i18n.German, 7, dartboard.Outer, "Äußeres Single 7"},
		{i18n.German, 7, dartboard.Double, "Doppel 7"},
		{i18n.German, 7, dartboard.Triple, "Triple 7"},
		{i18n.German, 25, dartboard.Bull, "Bull"},
		{i18n.German, 25, dartboard.Bullseye, "Bullseye"},
	}
	for _, tt := range tests {
		if got := fieldLabel(textsOf(tt.lang), tt.number, tt.segment); got != tt.want {
			t.Errorf("%s %d %s: got %q, want %q", tt.lang, tt.number, tt.segment, got, tt.want)
		}
	}
}
