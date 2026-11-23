package game

import (
	"testing"
	"time"

	"github.com/sj14/darts-o-mat/internal/server/domain/game/turn"
	"github.com/sj14/darts-o-mat/internal/server/domain/throw"
)

func TestTurnDeadlineAfterVisit(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	const pause, limit = 1500 * time.Millisecond, time.Minute
	g := Game{Settings: Settings{TurnSeconds: 60}}
	dart := func(player int64, offset time.Duration) throw.Dart {
		return throw.Dart{PlayerID: player, CreatedAt: at.Add(offset)}
	}
	up := func(player int64, darts int) turn.State {
		return turn.State{CurrentPlayerID: &player, DartsInTurn: darts}
	}

	tests := []struct {
		name    string
		entries []throw.Dart
		state   turn.State
		want    time.Time
	}{
		{"after another player's visit", []throw.Dart{dart(1, 0)}, up(2, 0), at.Add(pause + limit)},
		{"alone, the thrower goes on at once", []throw.Dart{dart(1, 0)}, up(1, 0), at.Add(limit)},
		{"in the middle of a visit", []throw.Dart{dart(2, 0), dart(1, time.Second)}, up(1, 1), at.Add(pause + limit)},
		{"a player left in the middle of a visit", []throw.Dart{dart(2, 0), dart(1, time.Second), {PlayerID: 3, Leave: true, CreatedAt: at.Add(2 * time.Second)}}, up(1, 1), at.Add(pause + limit)},
		{"a leave entry before the visit", []throw.Dart{dart(2, 0), {PlayerID: 3, Leave: true, CreatedAt: at.Add(time.Second)}}, up(1, 0), at.Add(time.Second + limit)},
		{"a leave entry in the visit's darts", []throw.Dart{dart(2, 0), dart(1, time.Second), {PlayerID: 3, Leave: true, CreatedAt: at.Add(2 * time.Second)}, dart(1, 3*time.Second)}, up(1, 2), at.Add(pause + limit)},
	}
	for _, tt := range tests {
		got, ok := g.TurnDeadline(tt.entries, tt.state, pause)
		if !ok || !got.Equal(tt.want) {
			t.Errorf("%s: deadline %v, %v; want %v", tt.name, got, ok, tt.want)
		}
	}
}
