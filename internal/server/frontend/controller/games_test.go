package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sj14/darts-o-mat/internal/common/dartboard"
	"github.com/sj14/darts-o-mat/internal/server/domain/throw"
)

func TestDartFromForm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		form url.Values
		want throw.Dart
		ok   bool
	}{
		{"inner single", url.Values{"number": {"20"}, "segment": {"inner"}}, throw.Dart{Number: 20, Segment: dartboard.Inner}, true},
		{"outer single", url.Values{"number": {"20"}, "segment": {"outer"}}, throw.Dart{Number: 20, Segment: dartboard.Outer}, true},
		{"keypad single", url.Values{"number": {"20"}, "segment": {"single"}}, throw.Dart{Number: 20, Segment: dartboard.Single}, true},
		{"double", url.Values{"number": {"20"}, "segment": {"double"}}, throw.Dart{Number: 20, Segment: dartboard.Double}, true},
		{"triple", url.Values{"number": {"20"}, "segment": {"triple"}}, throw.Dart{Number: 20, Segment: dartboard.Triple}, true},
		{"bull", url.Values{"number": {"25"}, "segment": {"bull"}}, throw.Dart{Number: 25, Segment: dartboard.Bull}, true},
		{"bullseye", url.Values{"number": {"25"}, "segment": {"bullseye"}}, throw.Dart{Number: 25, Segment: dartboard.Bullseye}, true},
		{"keypad single bull", url.Values{"number": {"25"}, "segment": {"single"}}, throw.Dart{Number: 25, Segment: dartboard.Bull}, true},
		{"keypad double bull", url.Values{"number": {"25"}, "segment": {"double"}}, throw.Dart{Number: 25, Segment: dartboard.Bullseye}, true},
		{"miss", url.Values{"miss": {"true"}}, throw.Dart{Segment: dartboard.Miss}, true},
		{"unknown segment", url.Values{"number": {"20"}, "segment": {"middle"}}, throw.Dart{}, false},
		{"no segment is left to validation", url.Values{"number": {"20"}}, throw.Dart{Number: 20, Segment: dartboard.NoSegment}, true},
		{"no number", url.Values{"segment": {"inner"}}, throw.Dart{}, false},
		{"number too big", url.Values{"number": {"300"}, "segment": {"inner"}}, throw.Dart{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodPost, "/games/1/darts", strings.NewReader(tt.form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			got, ok := dartFromForm(r)
			if ok != tt.ok || got != tt.want {
				t.Errorf("got %+v, %v, want %+v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}
