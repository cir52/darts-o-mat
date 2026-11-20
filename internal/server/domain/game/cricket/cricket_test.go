package cricket

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/sj14/darts-o-mat/internal/common/dartboard"
	"github.com/sj14/darts-o-mat/internal/server/domain/game/turn"
	"github.com/sj14/darts-o-mat/internal/server/domain/throw"
)

// hit is a dart entered by hand: number and multiplier (for the bull number, 1 is the
// bull and 2 the bullseye).
func hit(number, multiplier uint8) throw.Dart {
	segments := []dartboard.Segment{1: dartboard.Single, 2: dartboard.Double, 3: dartboard.Triple}
	if number == dartboard.BullNumber {
		segments = []dartboard.Segment{1: dartboard.Bull, 2: dartboard.Bullseye}
	}
	return throw.Dart{Number: number, Segment: segments[multiplier]}
}

var miss = throw.Dart{Segment: dartboard.Miss}

// misses returns a turn of three misses.
func misses() []throw.Dart { return []throw.Dart{miss, miss, miss} }

func darts(turns ...[]throw.Dart) []throw.Dart { return slices.Concat(turns...) }

// closeAll is a player's three turns closing every number, the last one after two darts.
var closeAll = [][]throw.Dart{
	{hit(20, 3), hit(19, 3), hit(18, 3)},
	{hit(17, 3), hit(16, 3), hit(15, 3)},
	{hit(dartboard.BullNumber, 2), hit(dartboard.BullNumber, 1)},
}

var standard = Settings{Variant: VariantStandard}
var cutThroat = Settings{Variant: VariantCutThroat}

func TestPlay(t *testing.T) {
	t.Parallel()
	const bull = dartboard.BullNumber
	tests := []struct {
		name        string
		settings    Settings
		darts       []throw.Dart
		wantMarks   [][NumTargets]uint8 // per player, in the order of Targets
		wantPoints  []int
		wantCurrent int64 // 0 means the game is over
		wantWinner  int64
	}{
		{
			name:        "no darts",
			settings:    standard,
			wantMarks:   [][NumTargets]uint8{{}, {}},
			wantPoints:  []int{0, 0},
			wantCurrent: 1,
		},
		{
			name:        "a triple closes the number",
			settings:    standard,
			darts:       []throw.Dart{hit(20, 3)},
			wantMarks:   [][NumTargets]uint8{{3}, {}},
			wantPoints:  []int{0, 0},
			wantCurrent: 1,
		},
		{
			name:        "marks beyond closing score while the opponent is open",
			settings:    standard,
			darts:       []throw.Dart{hit(20, 1), hit(20, 1), hit(20, 3)},
			wantMarks:   [][NumTargets]uint8{{3}, {}},
			wantPoints:  []int{40, 0},
			wantCurrent: 2,
		},
		{
			name:        "no points once the opponent has closed the number",
			settings:    standard,
			darts:       darts(misses(), []throw.Dart{hit(20, 3), miss, miss}, []throw.Dart{hit(20, 3), hit(20, 3)}),
			wantMarks:   [][NumTargets]uint8{{3}, {3}},
			wantPoints:  []int{0, 0},
			wantCurrent: 1,
		},
		{
			name:        "numbers below 15 and misses count nothing",
			settings:    standard,
			darts:       []throw.Dart{hit(14, 3), hit(1, 2), miss},
			wantMarks:   [][NumTargets]uint8{{}, {}},
			wantPoints:  []int{0, 0},
			wantCurrent: 2,
		},
		{
			name:        "bull counts one mark, bullseye two",
			settings:    standard,
			darts:       []throw.Dart{hit(bull, 1), hit(bull, 2), hit(bull, 2)},
			wantMarks:   [][NumTargets]uint8{{6: 3}, {}},
			wantPoints:  []int{50, 0}, // only the second bullseye scores
			wantCurrent: 2,
		},
		{
			name:        "cut-throat gives the points to the opponent",
			settings:    cutThroat,
			darts:       []throw.Dart{hit(20, 3), hit(20, 2)},
			wantMarks:   [][NumTargets]uint8{{3}, {}},
			wantPoints:  []int{0, 40},
			wantCurrent: 1,
		},
		{
			name:       "closing everything with equal points wins",
			settings:   standard,
			darts:      darts(closeAll[0], misses(), closeAll[1], misses(), closeAll[2]),
			wantMarks:  [][NumTargets]uint8{{3, 3, 3, 3, 3, 3, 3}, {}},
			wantPoints: []int{0, 0},
			wantWinner: 1,
		},
		{
			name:     "closing everything while behind doesn't win until the player is level",
			settings: standard,
			darts: darts(
				misses(), []throw.Dart{hit(20, 3), hit(20, 3), miss}, // player 2 scores 60
				[]throw.Dart{hit(20, 3), hit(19, 3), hit(18, 3)}, misses(),
				closeAll[1], misses(),
				[]throw.Dart{hit(bull, 2), hit(bull, 1), hit(19, 3)}, misses(), // closed, 57 against 60
			),
			wantMarks:   [][NumTargets]uint8{{3, 3, 3, 3, 3, 3, 3}, {3}},
			wantPoints:  []int{57, 60},
			wantCurrent: 1,
		},
		{
			name:        "cut-throat: closing everything with more points doesn't win",
			settings:    cutThroat,
			darts:       darts(misses(), []throw.Dart{hit(20, 3), hit(20, 1), miss}, closeAll[0], misses(), closeAll[1], misses(), closeAll[2]),
			wantMarks:   [][NumTargets]uint8{{3, 3, 3, 3, 3, 3, 3}, {3}},
			wantPoints:  []int{20, 0},
			wantCurrent: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st, err := Play(tt.settings, []int64{1, 2}, tt.darts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for i := range tt.wantMarks {
				if got := st.Players[i].Marks; got != tt.wantMarks[i] {
					t.Errorf("player %d marks = %v, want %v", i+1, got, tt.wantMarks[i])
				}
				if got := st.Players[i].Points; got != tt.wantPoints[i] {
					t.Errorf("player %d points = %d, want %d", i+1, got, tt.wantPoints[i])
				}
			}
			var current int64
			if st.CurrentPlayerID != nil {
				current = *st.CurrentPlayerID
			}
			if current != tt.wantCurrent {
				t.Errorf("current player = %d, want %d", current, tt.wantCurrent)
			}
			var winner int64
			if st.WinnerID != nil {
				winner = *st.WinnerID
			}
			if winner != tt.wantWinner || st.GameOver != (tt.wantWinner != 0) {
				t.Errorf("winner = %d (game over %v), want %d", winner, st.GameOver, tt.wantWinner)
			}
		})
	}
}

func TestCatchUpWins(t *testing.T) {
	t.Parallel()
	// Player 1 has closed everything with 57 against 60 (see TestPlay) and scores again.
	d := darts(
		misses(), []throw.Dart{hit(20, 3), hit(20, 3), miss},
		[]throw.Dart{hit(20, 3), hit(19, 3), hit(18, 3)}, misses(),
		closeAll[1], misses(),
		[]throw.Dart{hit(dartboard.BullNumber, 2), hit(dartboard.BullNumber, 1), hit(19, 3)}, misses(),
		[]throw.Dart{hit(19, 1)},
	)
	st, err := Play(standard, []int64{1, 2}, d)
	if err != nil {
		t.Fatal(err)
	}
	if !st.GameOver || *st.WinnerID != 1 || st.Players[0].Points != 76 {
		t.Errorf("over=%v points=%d, want player 1 to win with 76", st.GameOver, st.Players[0].Points)
	}
}

func TestCutThroatWinsWhenNotAhead(t *testing.T) {
	t.Parallel()
	// Player 2 scores 20 for player 1 first; player 1 closes everything with 20 against 0
	// and goes on scoring for player 2, until the dart that puts player 1 behind wins.
	d := darts(
		misses(), []throw.Dart{hit(20, 3), hit(20, 1), miss},
		closeAll[0], misses(),
		closeAll[1], misses(),
		[]throw.Dart{hit(dartboard.BullNumber, 2), hit(dartboard.BullNumber, 1), hit(19, 1)}, // 19 for player 2: 20 against 19
		misses(),
		[]throw.Dart{hit(19, 1)}, // 20 against 38: player 1 isn't ahead anymore
	)
	st, err := Play(cutThroat, []int64{1, 2}, d)
	if err != nil {
		t.Fatal(err)
	}
	if !st.GameOver || *st.WinnerID != 1 {
		t.Errorf("over=%v points=%d/%d, want player 1 to win", st.GameOver, st.Players[0].Points, st.Players[1].Points)
	}
}

func TestCutThroatWinsOnAnOpponentsDart(t *testing.T) {
	t.Parallel()
	// Player 1 closes everything with 60 points, more than player 2's 0. Player 3 then
	// scores for player 2 until player 1 isn't ahead of anyone: player 1 wins on
	// player 3's dart.
	d := darts(
		misses(), []throw.Dart{hit(20, 3), hit(20, 3), miss}, []throw.Dart{hit(20, 3), miss, miss},
		closeAll[0], misses(), misses(),
		closeAll[1], misses(), misses(),
		closeAll[2], []throw.Dart{miss}, misses(), []throw.Dart{hit(19, 3), hit(19, 3), hit(19, 1)},
	)
	st, err := Play(cutThroat, []int64{1, 2, 3}, d)
	if err != nil {
		t.Fatal(err)
	}
	points := []int{st.Players[0].Points, st.Players[1].Points, st.Players[2].Points}
	if !st.GameOver || *st.WinnerID != 1 || !slices.Equal(points, []int{60, 76, 60}) {
		t.Errorf("over=%v points=%v, want player 1 to win with 60 against 76 and 60", st.GameOver, points)
	}
	if last := st.Turns[len(st.Turns)-1]; last.PlayerID != 3 || !last.Complete {
		t.Errorf("last turn by player %d (complete %v), want player 3's, complete", last.PlayerID, last.Complete)
	}
}

func TestLeavingDecidesTheLeg(t *testing.T) {
	t.Parallel()
	// Player 3 leads with 60, player 1 has closed everything with 0. When player 3
	// leaves, player 1 is level with player 2 and wins without another dart.
	log := turn.Log{
		PlayerIDs: []int64{1, 2, 3},
		Entries: darts(
			misses(), misses(), []throw.Dart{hit(20, 3), hit(20, 3), miss},
			closeAll[0], misses(), misses(),
			closeAll[1], misses(), misses(),
			closeAll[2], []throw.Dart{miss},
			[]throw.Dart{{Leave: true, PlayerID: 3}},
		),
	}
	st, err := PlayLog(standard, []int64{1, 2, 3}, log)
	if err != nil {
		t.Fatal(err)
	}
	if !st.GameOver || *st.WinnerID != 1 {
		t.Fatalf("over=%v, want player 1 to win when player 3 leaves", st.GameOver)
	}
	if got := st.Players[0].BestLeg; got != 12 {
		t.Errorf("best leg = %d darts, want 12 (no dart after the leaving)", got)
	}
	if got := len(st.Times.Legs); got != 1 {
		t.Errorf("%d leg times, want 1", got)
	}
}

func TestCutThroatPointsToEveryOpenOpponent(t *testing.T) {
	t.Parallel()
	d := darts(misses(), []throw.Dart{hit(20, 3), miss, miss}, misses(), []throw.Dart{hit(20, 3), hit(20, 2)})
	st, err := Play(cutThroat, []int64{1, 2, 3}, d)
	if err != nil {
		t.Fatal(err)
	}
	if got := []int{st.Players[0].Points, st.Players[1].Points, st.Players[2].Points}; !slices.Equal(got, []int{0, 0, 40}) {
		t.Errorf("points = %v, want only player 3, who hasn't closed the 20, to get them", got)
	}

	d = []throw.Dart{hit(20, 3), hit(20, 3)}
	st, err = Play(cutThroat, []int64{1, 2, 3}, d)
	if err != nil {
		t.Fatal(err)
	}
	if got := []int{st.Players[1].Points, st.Players[2].Points}; !slices.Equal(got, []int{60, 60}) {
		t.Errorf("points = %v, want 60 each", got)
	}
	if got := st.Turns[0].Points; got != 120 {
		t.Errorf("turn points = %d, want 120 (60 for each opponent)", got)
	}
}

func TestPlayersWhoLeftDontCount(t *testing.T) {
	t.Parallel()
	// Player 3 leaves; player 2 closes the 20, so player 1's extra marks on it score nothing.
	log := turn.Log{
		PlayerIDs: []int64{1, 2, 3},
		Entries: darts(
			[]throw.Dart{{Leave: true, PlayerID: 3}},
			misses(),
			[]throw.Dart{hit(20, 3), miss, miss},
			[]throw.Dart{hit(20, 3), hit(20, 3)},
		),
	}
	st, err := PlayLog(standard, []int64{1, 2, 3}, log)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Players[0].Points; got != 0 {
		t.Errorf("points = %d, want 0", got)
	}
	if got := st.Players[0].Stats.Marks; got != 3 {
		t.Errorf("marks = %d, want 3 (the marks beyond closing score nothing)", got)
	}
}

func TestDead(t *testing.T) {
	t.Parallel()
	// Player 1 closes the 20, player 2 the 20 and the 19, player 3 leaves without closing anything.
	log := turn.Log{
		PlayerIDs: []int64{1, 2, 3},
		Entries: darts(
			[]throw.Dart{{Leave: true, PlayerID: 3}},
			[]throw.Dart{hit(20, 3), miss, miss},
			[]throw.Dart{hit(20, 3), hit(19, 3), miss},
		),
	}
	st, err := PlayLog(standard, []int64{1, 2, 3}, log)
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range []bool{true, false, false, false, false, false, false} {
		if got := st.Dead(k); got != want {
			t.Errorf("Dead(%d) (%d) = %v, want %v", k, Targets[k], got, want)
		}
	}
}

func TestPlayAfterWin(t *testing.T) {
	t.Parallel()
	_, err := Play(standard, []int64{1}, darts(closeAll[0], closeAll[1], closeAll[2], []throw.Dart{miss}))
	if !errors.Is(err, turn.ErrGameOver) {
		t.Errorf("got %v, want ErrGameOver", err)
	}
}

func TestStats(t *testing.T) {
	t.Parallel()
	// Player 1: 3 + 3 (scoring) + 0 marks in three darts; player 2: one dart, 1 mark.
	st, err := Play(standard, []int64{1, 2}, []throw.Dart{hit(20, 3), hit(20, 3), hit(5, 1), hit(19, 1)})
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Players[0].Stats; got.Darts != 3 || got.Marks != 6 || got.MPR != 6 {
		t.Errorf("player 1 stats = %+v, want 3 darts, 6 marks, MPR 6", got)
	}
	if got := st.Players[1].Stats; got.Darts != 1 || got.Marks != 1 || got.MPR != 3 {
		t.Errorf("player 2 stats = %+v, want 1 dart, 1 mark, MPR 3", got)
	}
}

func TestTurns(t *testing.T) {
	t.Parallel()
	d := []throw.Dart{hit(20, 3), hit(20, 1), hit(5, 1), hit(19, 2)}
	st, err := Play(standard, []int64{1, 2}, d)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Turns) != 2 {
		t.Fatalf("got %d turns, want 2", len(st.Turns))
	}
	first := st.Turns[0]
	if first.PlayerID != 1 || first.Points != 20 || !first.Complete || !slices.Equal(first.Marks, []int{3, 1, 0}) {
		t.Errorf("first turn = %+v", first)
	}
	second := st.Turns[1]
	if second.PlayerID != 2 || second.Complete || !slices.Equal(second.Marks, []int{2}) {
		t.Errorf("turn in progress = %+v", second)
	}

	// On a board the turn is over while waiting for the takeout.
	st, _ = PlayLog(standard, []int64{1, 2}, turn.Log{Entries: d[:3], WaitForTakeout: true})
	if !st.WaitingForTakeout || !st.Turns[0].Complete {
		t.Errorf("waiting=%v complete=%v, want a complete turn waiting for the takeout", st.WaitingForTakeout, st.Turns[0].Complete)
	}

	// Winning ends the turn early.
	st, _ = Play(standard, []int64{1}, darts(closeAll...))
	if last := st.Turns[len(st.Turns)-1]; !last.Complete || len(last.Darts) != 2 {
		t.Errorf("last turn = %+v, want complete after 2 darts", last)
	}
}

func TestLegs(t *testing.T) {
	t.Parallel()
	log := turn.Log{Legs: 2, Entries: darts(closeAll[0], misses(), closeAll[1], misses(), closeAll[2])}
	st, err := PlayLog(standard, []int64{1, 2}, log)
	if err != nil {
		t.Fatal(err)
	}
	if st.GameOver || st.Leg != 2 || st.Players[0].Legs != 1 || st.Players[0].Marks != [NumTargets]uint8{} || *st.CurrentPlayerID != 2 {
		t.Errorf("after leg 1: over=%v leg=%d legs=%d marks=%v, want leg 2 started by player 2 with the marks reset",
			st.GameOver, st.Leg, st.Players[0].Legs, st.Players[0].Marks)
	}
	if got := st.Players[0].BestLeg; got != 8 {
		t.Errorf("best leg = %d darts, want 8", got)
	}
}

func TestSettings(t *testing.T) {
	t.Parallel()
	if err := (Settings{}).Validate(); err == nil {
		t.Error("settings without a variant are valid, want an error")
	}
	for _, s := range []Settings{standard, cutThroat} {
		if err := s.Validate(); err != nil {
			t.Errorf("%+v: %v", s, err)
		}
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		var got Settings
		if err := json.Unmarshal(b, &got); err != nil || got != s {
			t.Errorf("%s decodes to %+v (%v), want %+v", b, got, err, s)
		}
	}
	if err := json.Unmarshal([]byte(`{"variant":"no-score"}`), &Settings{}); err == nil {
		t.Error("unknown variant decodes, want an error")
	}
}

func TestHit(t *testing.T) {
	t.Parallel()
	// Player 1 closes the 20, hits the 14 and then the 20 again while player 2 has it open.
	d := []throw.Dart{hit(20, 3), hit(14, 1), hit(20, 1)}
	for i := range d {
		d[i].ID = int64(i + 1)
	}
	st, err := Play(standard, []int64{1, 2}, d)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		id      int64
		hit, ok bool
	}{
		{1, true, true},
		{2, false, true},
		{3, true, true},   // scores
		{4, false, false}, // not thrown
	} {
		if hit, ok := st.Hit(tt.id); hit != tt.hit || ok != tt.ok {
			t.Errorf("Hit(%d) = %v, %v; want %v, %v", tt.id, hit, ok, tt.hit, tt.ok)
		}
	}
}
