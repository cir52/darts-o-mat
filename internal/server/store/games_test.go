package store

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/sj14/darts-o-mat/internal/common/dartboard"
	"github.com/sj14/darts-o-mat/internal/server/domain/board"
	"github.com/sj14/darts-o-mat/internal/server/domain/game"
	"github.com/sj14/darts-o-mat/internal/server/domain/game/x01"
	"github.com/sj14/darts-o-mat/internal/server/domain/player"
	"github.com/sj14/darts-o-mat/internal/server/domain/throw"
)

// Of the games on a board only those that may still be running are listed: started, not
// aborted, not completed. Games on other boards aren't.
func TestListUnfinishedGamesByBoard(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st, err := NewSQLite(ctx, "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() }) // the in-memory database is gone with it, also for -count
	b, other := board.Board{AccountID: 1, Name: "B"}, board.Board{AccountID: 1, Name: "Other"}
	for _, x := range []*board.Board{&b, &other} {
		if err := st.CreateBoard(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	create := func(boardID int64, change func(g *game.Game) error) int64 {
		g := game.Game{
			Settings:     game.Settings{Mode: game.ModeX01, Legs: 1, Sets: 1, Rules: &x01.Settings{StartScore: 501, OutMode: x01.OutModeDouble}},
			Participants: []game.Participant{{PlayerID: 1, BoardID: &boardID}},
		}
		if err := st.CreateGame(ctx, &g); err != nil {
			t.Fatal(err)
		}
		if change != nil {
			if err := change(&g); err != nil {
				t.Fatal(err)
			}
		}
		return g.ID
	}
	start := func(g *game.Game) error { g.StartedAt = &now; return st.StartGame(ctx, g) }

	running := create(b.ID, start)
	create(b.ID, nil) // in the lobby
	create(b.ID, func(g *game.Game) error {
		if err := start(g); err != nil {
			return err
		}
		g.AbortedAt = &now
		return st.AbortGame(ctx, g)
	})
	create(b.ID, func(g *game.Game) error {
		if err := start(g); err != nil {
			return err
		}
		g.CompletedAt = &now
		return st.CompleteGame(ctx, g)
	})
	create(other.ID, start)

	games, err := st.ListUnfinishedGamesByBoard(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 || games[0].ID != running {
		t.Errorf("listed %d games, want only the running one (%d): %+v", len(games), running, games)
	}
}

// Darts with times with and without fractions of a second are found alike: a dart since a
// time is found among both, since SQLite compares the texts of the times.
func TestListGamesWithDartsSinceMixedTimes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st, err := NewSQLite(ctx, "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	anna := player.Player{Name: "Anna"}
	if err := st.StorePlayer(ctx, &anna); err != nil {
		t.Fatal(err)
	}
	create := func() int64 {
		g := game.Game{
			Settings:     game.Settings{Mode: game.ModeX01, Legs: 1, Sets: 1, Rules: &x01.Settings{StartScore: 501, OutMode: x01.OutModeDouble}},
			Participants: []game.Participant{{PlayerID: anna.ID}},
		}
		if err := st.CreateGame(ctx, &g); err != nil {
			t.Fatal(err)
		}
		return g.ID
	}
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	old, fresh := create(), create()
	if err := st.AddDart(ctx, &throw.Dart{GameID: old, PlayerID: anna.ID, Number: 20, Segment: dartboard.Single}); err != nil {
		t.Fatal(err)
	}
	// The database's own format: whole seconds, no zone.
	if _, err := st.db.ExecContext(ctx, "UPDATE darts SET created_at = '2026-01-01 12:00:00' WHERE game_id = ?", old); err != nil {
		t.Fatal(err)
	}
	if err := st.AddDart(ctx, &throw.Dart{GameID: fresh, PlayerID: anna.ID, Number: 20, Segment: dartboard.Single, CreatedAt: at.Add(500 * time.Millisecond)}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		since time.Time
		want  []int64
	}{
		{at.Add(-100 * time.Millisecond), []int64{old, fresh}},
		{at, []int64{fresh}},
		{at.Add(400 * time.Millisecond), []int64{fresh}},
		{at.Add(600 * time.Millisecond), nil},
		{at.Add(time.Second), nil},
	}
	for _, tt := range tests {
		games, err := st.ListGamesWithDartsSince(ctx, 1, anna.ID, tt.since)
		if err != nil {
			t.Fatal(err)
		}
		var got []int64
		for _, g := range games {
			got = append(got, g.ID)
		}
		slices.Sort(got)
		if !slices.Equal(got, tt.want) {
			t.Errorf("games with darts since %v: %v, want %v", tt.since.Format(time.RFC3339Nano), got, tt.want)
		}
	}
}
