package throw

import (
	"encoding/json"
	"testing"

	"github.com/sj14/darts-o-mat/internal/common/dartboard"
)

func TestJSON(t *testing.T) {
	t.Parallel()
	// A takeout has no segment, and none in its JSON.
	if data, _ := json.Marshal(Dart{Takeout: true}); string(data) != `{"id":0,"number":0,"takeout":true}` {
		t.Errorf("takeout: %s", data)
	}
}

// The log entry scores like the hit it holds.
func TestHit(t *testing.T) {
	t.Parallel()
	d := FromHit(dartboard.Hit{Number: 20, Segment: dartboard.Triple})
	if d.Score() != 60 || d.String() != "T20" || d.Validate() != nil || d.Hit() != (dartboard.Hit{Number: 20, Segment: dartboard.Triple}) {
		t.Errorf("%+v: %d points, %q, %v", d, d.Score(), d.String(), d.Validate())
	}
}

func TestFirstOfLast(t *testing.T) {
	t.Parallel()
	a, leave, takeout := Dart{}, Dart{Leave: true}, Dart{Takeout: true}
	log := []Dart{a, leave, a, takeout, a}
	for n, want := range map[int]int{0: 5, 1: 4, 2: 2, 3: 0, 4: 0} {
		if got := FirstOfLast(log, n); got != want {
			t.Errorf("FirstOfLast(%d) = %d, want %d", n, got, want)
		}
	}
	if got := FirstOfLast(nil, 3); got != 0 {
		t.Errorf("without a log: %d, want 0", got)
	}
}
