package view

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sj14/darts-o-mat/internal/common/dartboard"
	"github.com/sj14/darts-o-mat/internal/common/i18n"
	"github.com/sj14/darts-o-mat/internal/server/domain/board"
	"github.com/sj14/darts-o-mat/internal/server/domain/game"
	"github.com/sj14/darts-o-mat/internal/server/domain/game/cricket"
	"github.com/sj14/darts-o-mat/internal/server/domain/game/turn"
	"github.com/sj14/darts-o-mat/internal/server/domain/game/x01"
	"github.com/sj14/darts-o-mat/internal/server/domain/throw"
	"github.com/sj14/darts-o-mat/internal/server/service"
)

func TestDefaultBoard(t *testing.T) {
	t.Parallel()
	view := func(id int64, used time.Duration) board.Board {
		b := board.Board{ID: id}
		if used != 0 {
			at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(used)
			b.LastUsedAt = &at
		}
		return b
	}
	tests := []struct {
		name   string
		boards []board.Board
		want   int64
	}{
		{"no boards", nil, 0},
		{"never used: the first", []board.Board{view(3, 0), view(5, 0)}, 3},
		{"the last used", []board.Board{view(3, time.Hour), view(5, 2*time.Hour), view(7, 0)}, 5},
		{"a used one over unused ones before it", []board.Board{view(3, 0), view(5, time.Hour)}, 5},
	}
	for _, tt := range tests {
		if got := defaultBoard(tt.boards); got != tt.want {
			t.Errorf("%s: defaultBoard = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestHandEntry(t *testing.T) {
	t.Parallel()
	board1, board2 := int64(1), int64(2)
	const anna, ben, guest = int64(10), int64(20), int64(30)
	progress := func(current int64) progress {
		return progress{State: turn.State{CurrentPlayerID: &current}}
	}
	players := func(boards ...*int64) service.GameView {
		ids := []int64{anna, guest, ben}
		v := service.GameView{}
		for i, b := range boards {
			v.Game.Participants = append(v.Game.Participants, game.Participant{PlayerID: ids[i], Status: game.InvitationAccepted, BoardID: b})
		}
		return v
	}
	bot := func(v service.GameView) service.GameView { // the guest is a bot
		v.Game.Participants[1].BotSpread = 20
		return v
	}
	with := func(v service.GameView, mine ...int64) service.GameView {
		v.Mine = mine
		return v
	}

	tests := []struct {
		name      string
		v         service.GameView
		current   int64
		hand, add bool
	}{
		{"no board, my turn", with(players(nil, nil, nil), anna, guest), guest, true, false},
		{"no board, a friend's turn", with(players(nil, nil, nil), anna, guest), ben, false, false},
		{"at a board, my turn", with(players(&board1, &board1, nil), anna, guest), guest, false, true},
		{"on my own board", with(players(nil, nil, &board2), ben), ben, false, true},
		{"joined without a board next to a host with one", with(players(&board1, &board1, nil), ben), ben, true, false},
		{"watching a friend who has a board", with(players(nil, nil, &board2), anna, guest), ben, false, false},
		{"my bot's turn without boards", with(bot(players(nil, nil, nil)), anna, guest), guest, true, false},
		{"my bot's turn next to my board", with(bot(players(&board1, nil, nil)), anna, guest), guest, false, false},
	}
	for _, tt := range tests {
		tt.v.CanThrow = true
		p := progress(tt.current)
		if got := handEntry(tt.v, p); got != tt.hand {
			t.Errorf("%s: handEntry = %v, want %v", tt.name, got, tt.hand)
		}
		if got := mayAddDart(tt.v, p); got != tt.add {
			t.Errorf("%s: mayAddDart = %v, want %v", tt.name, got, tt.add)
		}
	}

	// An account whose player was skipped out doesn't enter darts for its bot.
	left := with(bot(players(nil, nil, nil)), anna, guest)
	if handEntry(left, progress(guest)) {
		t.Error("keypad for the bot of an account that can't throw anymore")
	}
}

func TestBoardAction(t *testing.T) {
	t.Parallel()
	board := int64(1)
	const anna, ben, robo = int64(10), int64(20), int64(30)
	v := service.GameView{CanThrow: true, Mine: []int64{anna, robo}}
	v.Game.Participants = []game.Participant{
		{PlayerID: anna, Status: game.InvitationAccepted},
		{PlayerID: ben, Status: game.InvitationAccepted, BoardID: &board},
		{PlayerID: robo, Status: game.InvitationAccepted, BotSpread: 20},
	}
	withCamera := v
	withCamera.Mine = []int64{ben}
	turnOf := func(current int64) progress {
		return progress{State: turn.State{CurrentPlayerID: &current}}
	}
	bullOff := turnOf(anna)
	bullOff.BullingOff = true
	takeout := turnOf(anna)
	takeout.WaitingForTakeout = true

	watching := v
	watching.CanThrow = false
	const enter, correct = "/games/0/darts", "/games/0/darts/7"
	dart := &throw.Dart{ID: 7}

	tests := []struct {
		name    string
		v       service.GameView
		p       progress
		editing *throw.Dart
		want    string
	}{
		{"my turn by hand", v, turnOf(anna), nil, enter},
		{"my turn with cameras", withCamera, turnOf(ben), nil, enter},
		{"another account's turn", v, turnOf(ben), nil, ""},
		{"my bot's turn", v, turnOf(robo), nil, ""},
		{"bull-off", v, bullOff, nil, ""},
		{"takeout pending", v, takeout, nil, ""},
		{"correcting a dart", v, turnOf(ben), dart, correct},
		{"correcting in the bull-off", v, bullOff, dart, correct},
		{"correcting without being able to throw", watching, turnOf(anna), dart, ""},
	}
	for _, tt := range tests {
		if got := boardAction(tt.v, tt.p, tt.editing); got != tt.want {
			t.Errorf("%s: boardAction = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLastTurnOf(t *testing.T) {
	t.Parallel()
	st := x01.State{Turns: []x01.Turn{
		{PlayerID: 1, Round: 1, Points: 125, Complete: true},
		{PlayerID: 2, Round: 1, Points: 0, Bust: true, Complete: true},
		{PlayerID: 1, Round: 2, Points: 60, Complete: true},
		{PlayerID: 2, Round: 2, Points: 20}, // still throwing
	}}
	v := service.GameView{State: st}
	texts := textsOf(i18n.English)

	if got, ok := lastTurnOf(texts, v, 1); !ok || got.Value != "60" || got.Strong || got.Bust {
		t.Errorf("player 1 = %+v, %v; want the 60", got, ok)
	}
	if got, ok := lastTurnOf(texts, v, 2); !ok || !got.Bust || got.Value != texts.Busted() {
		t.Errorf("player 2 = %+v, %v; want the bust (the unfinished turn doesn't count)", got, ok)
	}
	if _, ok := lastTurnOf(texts, v, 3); ok {
		t.Error("a player who didn't throw has a last turn")
	}
}

// visitView is an X01 game of players 1 and 2 after the darts, the last of them thrown at
// the given time.
func visitView(t *testing.T, s x01.Settings, log turn.Log, last time.Time) service.GameView {
	t.Helper()
	return visitViewOf(t, s, []int64{1, 2}, log, last)
}

func visitViewOf(t *testing.T, s x01.Settings, players []int64, log turn.Log, last time.Time) service.GameView {
	t.Helper()
	log.PlayerIDs = players
	st, err := x01.PlayLog(s, players, log)
	if err != nil {
		t.Fatal(err)
	}
	darts := slices.Clone(log.Entries)
	darts[len(darts)-1].CreatedAt = last
	return service.GameView{State: st, Darts: darts}
}

func TestProgressAtPausesAfterVisit(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	single := func(id int64) throw.Dart {
		return throw.Dart{ID: id, PlayerID: 1, Number: 20, Segment: dartboard.Single}
	}
	s := x01.Settings{StartScore: 501, OutMode: x01.OutModeDouble}
	three := turn.Log{Entries: []throw.Dart{single(1), single(2), single(3)}}

	v := visitView(t, s, three, at)
	p := progressAt(v, at.Add(500*time.Millisecond))
	if !p.Paused || p.CurrentPlayerID == nil || *p.CurrentPlayerID != 1 || p.DartsInTurn != 3 || p.PauseLeft != service.VisitPause-500*time.Millisecond {
		t.Errorf("right after the third dart: %+v, want player 1 with 3 darts paused for the rest", p)
	}
	if got := turnDarts(v, p); len(got) != 3 {
		t.Errorf("the visit shows %d darts, want 3", len(got))
	}
	if p := progressAt(v, at.Add(service.VisitPause)); p.Paused || *p.CurrentPlayerID != 2 || p.DartsInTurn != 0 {
		t.Errorf("after the pause: %+v, want player 2 up", p)
	}

	two := visitView(t, s, turn.Log{Entries: three.Entries[:2]}, at)
	if p := progressAt(two, at); p.Paused || *p.CurrentPlayerID != 1 {
		t.Errorf("after a dart that doesn't finish the visit: %+v, want no pause", p)
	}

	board := visitView(t, s, turn.Log{Entries: three.Entries, WaitForTakeout: true}, at)
	if p := progressAt(board, at); p.Paused || !p.WaitingForTakeout {
		t.Errorf("waiting for the takeout: %+v, want no pause", p)
	}

	won := visitView(t, x01.Settings{StartScore: 20}, turn.Log{Entries: three.Entries[:1]}, at)
	if p := progressAt(won, at); p.Paused || !p.GameOver {
		t.Errorf("at the game's end: %+v, want no pause", p)
	}

	leg := visitView(t, x01.Settings{StartScore: 20}, turn.Log{Entries: three.Entries[:1], Legs: 2}, at)
	if p := progressAt(leg, at); p.Paused || p.LegWinner == nil || *p.CurrentPlayerID != 2 {
		t.Errorf("after a won leg: %+v, want the next leg's player up at once", p)
	}
}

func TestProgressAtEdgeCases(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := x01.Settings{StartScore: 501, OutMode: x01.OutModeDouble}
	single := func(id, player int64) throw.Dart {
		return throw.Dart{ID: id, PlayerID: player, Number: 20, Segment: dartboard.Single}
	}

	// A player who left in the middle of a visit: the entry isn't a dart of it.
	leave := throw.Dart{ID: 2, PlayerID: 3, Leave: true}
	left := visitViewOf(t, s, []int64{1, 2, 3}, turn.Log{Entries: []throw.Dart{single(1, 1), leave, single(3, 1), single(4, 1)}}, at)
	p := progressAt(left, at)
	if !p.Paused || p.DartsInTurn != 3 {
		t.Fatalf("a visit with a leaving player in it: %+v, want it paused with 3 darts", p)
	}
	if got := turnDarts(left, p); len(got) != 3 || got[0].ID != 1 || got[1].ID != 3 || got[2].ID != 4 {
		t.Errorf("the visit shows %+v, want the three darts without the leave entry", got)
	}

	// The second visit of the other player shows its own round.
	log := turn.Log{}
	for i := range 6 {
		log.Entries = append(log.Entries, single(int64(i+1), int64(i/3+1)))
	}
	second := visitView(t, s, log, at)
	if p := progressAt(second, at); !p.Paused || p.Round != 1 || *p.CurrentPlayerID != 2 {
		t.Errorf("player 2's visit of round 1: %+v, want it paused in round 1", p)
	}
	if p := progressAt(second, at.Add(service.VisitPause)); p.Paused || p.Round != 2 {
		t.Errorf("after the pause: %+v, want round 2", p)
	}

	// Alone (or the only one left) the thrower goes on at once.
	solo := visitViewOf(t, s, []int64{1}, turn.Log{Entries: []throw.Dart{single(1, 1), single(2, 1), single(3, 1)}}, at)
	if p := progressAt(solo, at); p.Paused || *p.CurrentPlayerID != 1 {
		t.Errorf("a visit alone: %+v, want no pause", p)
	}

	// An entry from the future (the clock of another replica is ahead) doesn't pause.
	three := visitView(t, s, turn.Log{Entries: []throw.Dart{single(1, 1), single(2, 1), single(3, 1)}}, at)
	if p := progressAt(three, at.Add(-time.Second)); p.Paused || *p.CurrentPlayerID != 2 {
		t.Errorf("an entry from the future: %+v, want no pause", p)
	}
	if _, ok := visitScore(textsOf(i18n.English), three, progressOf(three), at.Add(-time.Second)); ok {
		t.Error("an entry from the future shows its score")
	}

	// The pause only applies where the page can end it.
	if p := pageProgress(three, nil, false, ""); p.Paused {
		t.Errorf("an old visit is paused: %+v", p) // 2026 is long ago
	}
	for _, tt := range []struct {
		editing *throw.Dart
		adding  bool
		errMsg  string
	}{{&throw.Dart{ID: 1}, false, ""}, {nil, true, ""}, {nil, false, "error"}} {
		fresh := three
		fresh.Darts = slices.Clone(three.Darts)
		fresh.Darts[2].CreatedAt = time.Now()
		if !pageProgress(fresh, nil, false, "").Paused {
			t.Fatal("a fresh visit isn't paused")
		}
		if p := pageProgress(fresh, tt.editing, tt.adding, tt.errMsg); p.Paused || *p.CurrentPlayerID != 2 {
			t.Errorf("%+v: %+v, want no pause", tt, p)
		}
	}
}

func TestPreviousTurnSkipsLeaveEntries(t *testing.T) {
	t.Parallel()
	single := func(id, player int64) throw.Dart {
		return throw.Dart{ID: id, PlayerID: player, Number: 20, Segment: dartboard.Single}
	}
	log := turn.Log{Entries: []throw.Dart{single(1, 1), {ID: 2, PlayerID: 3, Leave: true}, single(3, 1), single(4, 1)}}
	v := visitViewOf(t, x01.Settings{StartScore: 501, OutMode: x01.OutModeDouble}, []int64{1, 2, 3}, log, time.Now().Add(-time.Hour))
	got := previousTurn(v)
	if len(got) != 3 || got[0].ID != 1 || got[1].ID != 3 || got[2].ID != 4 {
		t.Errorf("the previous turn is %+v, want the three darts of player 1", got)
	}
	if got := boardDarts(v, progressOf(v)); len(got) != 3 || !got[0].Faded {
		t.Errorf("the board marks %+v, want the three darts faded", got)
	}
}

// The page of a paused visit carries the request that ends the pause, and none where the
// page shows no pause (an error): then nothing would end it.
func TestScoresRefetchesAfterPause(t *testing.T) {
	t.Parallel()
	single := func(id int64) throw.Dart {
		return throw.Dart{ID: id, PlayerID: 1, Number: 20, Segment: dartboard.Single}
	}
	rules := x01.Settings{StartScore: 501, OutMode: x01.OutModeDouble}
	v := visitView(t, rules, turn.Log{Entries: []throw.Dart{single(1), single(2), single(3)}}, time.Now())
	v.Game.ID = 7
	v.Game.Mode, v.Game.Rules = game.ModeX01, &rules
	ctx := i18n.With(context.Background(), i18n.English)
	render := func(errMsg string) string {
		t.Helper()
		var b strings.Builder
		if err := scores(v, errMsg, nil, false).Render(ctx, &b); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	if got := render(""); !strings.Contains(got, `hx-trigger="load delay:`) || !strings.Contains(got, `hx-get="/games/7/board"`) {
		t.Error("a paused visit has no request that fetches the next player")
	}
	if got := render("something went wrong"); strings.Contains(got, `hx-trigger="load delay:`) {
		t.Error("the page shows an error and no pause, but fetches the next player")
	}
}

func TestPauseTrigger(t *testing.T) {
	t.Parallel()
	if got, want := pauseTrigger(progress{PauseLeft: 1200 * time.Millisecond}), "load delay:1201ms, every 2s"; got != want {
		t.Errorf("pauseTrigger = %q, want %q", got, want)
	}
}

func TestVisitScore(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dart := func(id int64, segment dartboard.Segment, number uint8) throw.Dart {
		return throw.Dart{ID: id, PlayerID: 1, Number: number, Segment: segment}
	}
	s := x01.Settings{StartScore: 40, OutMode: x01.OutModeDouble}
	texts := textsOf(i18n.English)
	score := func(v service.GameView, now time.Time) (lastTurn, bool) {
		return visitScore(texts, v, progressAt(v, now), now)
	}

	v := visitView(t, s, turn.Log{Entries: []throw.Dart{dart(1, dartboard.Single, 5), dart(2, dartboard.Single, 5), dart(3, dartboard.Single, 5)}}, at)
	if got, ok := score(v, at); !ok || got.Value != "15" || got.Bust {
		t.Errorf("a visit = %+v, %v; want the 15", got, ok)
	}
	if _, ok := score(v, at.Add(service.VisitPause)); ok {
		t.Error("the score shows after the pause: a later render keeps the faded score in the page")
	}
	fourth := dart(4, dartboard.Single, 5)
	fourth.PlayerID = 2
	next := visitView(t, s, turn.Log{Entries: append(slices.Clone(v.Darts), fourth)}, at)
	if _, ok := score(next, at); ok {
		t.Error("the score of the visit shows after the next player's dart")
	}

	bust := visitView(t, s, turn.Log{Entries: []throw.Dart{dart(1, dartboard.Single, 20), dart(2, dartboard.Single, 20)}}, at)
	if got, ok := score(bust, at); !ok || !got.Bust || got.Value != texts.Busted() {
		t.Errorf("a bust = %+v, %v; want Busted", got, ok)
	}

	big := visitView(t, x01.Settings{StartScore: 501}, turn.Log{Entries: []throw.Dart{dart(1, dartboard.Triple, 20), dart(2, dartboard.Triple, 20), dart(3, dartboard.Triple, 20)}}, at)
	if _, ok := score(big, at); ok {
		t.Error("a 180 has its own effect")
	}

	won := visitView(t, x01.Settings{StartScore: 100, OutMode: x01.OutModeDouble}, turn.Log{Entries: []throw.Dart{dart(1, dartboard.Triple, 20), dart(2, dartboard.Double, 20)}, Legs: 2}, at)
	if _, ok := score(won, at); ok {
		t.Error("a won leg has its own banner")
	}

	board := visitView(t, s, turn.Log{Entries: []throw.Dart{dart(1, dartboard.Single, 5), dart(2, dartboard.Single, 5), dart(3, dartboard.Single, 5)}, WaitForTakeout: true}, at)
	if got, ok := score(board, at); !ok || got.Value != "15" {
		t.Errorf("waiting for the takeout = %+v, %v; want the 15", got, ok)
	}
	if _, ok := score(board, at.Add(time.Hour)); ok {
		t.Error("an hour after the dart the score still shows")
	}
}

func TestEffectStyleAt(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		now  time.Time
		want string
	}{
		{at.Add(1250 * time.Millisecond), "animation-delay:-1.250s"},
		{at, "animation-delay:-0.000s"},
		{at.Add(-time.Second), "animation-delay:-0.000s"}, // a clock that is behind
	}
	for _, tt := range tests {
		if got := effectStyleAt(at, tt.now); string(got) != tt.want {
			t.Errorf("effectStyleAt(%v) = %q, want %q", tt.now.Sub(at), got, tt.want)
		}
	}
}

// The banner of a won leg counts from the dart that won it, also after the takeout.
func TestLegEndStyle(t *testing.T) {
	t.Parallel()
	at := time.Now().Add(-10 * time.Second)
	won := throw.Dart{ID: 1, PlayerID: 1, Number: 20, Segment: dartboard.Single, CreatedAt: at}
	takeout := throw.Dart{ID: 2, Takeout: true}
	v := visitView(t, x01.Settings{StartScore: 20}, turn.Log{Entries: []throw.Dart{won, takeout}, Legs: 2, WaitForTakeout: true}, time.Now())
	p := progressOf(v)
	if p.LegWinner == nil {
		t.Fatalf("after the takeout: %+v, want the leg's winner still shown", p)
	}
	style := string(legEndStyle(v, p))
	var delay float64
	if _, err := fmt.Sscanf(style, "animation-delay:-%fs", &delay); err != nil || delay < 10 || delay > 11 {
		t.Errorf("legEndStyle = %q, want about 10 s since the winning dart", style)
	}
}

func TestCricketMark(t *testing.T) {
	t.Parallel()
	for marks, want := range []struct{ lines, circles int }{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 1}} {
		var b strings.Builder
		if err := cricketMark(uint8(marks)).Render(context.Background(), &b); err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(b.String(), "<line "); got != want.lines {
			t.Errorf("%d marks draw %d lines, want %d", marks, got, want.lines)
		}
		if got := strings.Count(b.String(), "<circle "); got != want.circles {
			t.Errorf("%d marks draw %d circles, want %d", marks, got, want.circles)
		}
	}
}

func TestCricketDartsWithoutMarksMiss(t *testing.T) {
	t.Parallel()
	scored := throw.Dart{ID: 1, PlayerID: 1, Number: 20, Segment: dartboard.Triple}
	dead := throw.Dart{ID: 2, PlayerID: 1, Number: 20, Segment: dartboard.Single}
	// Alone in the game, the player's triple closes the 20 and nothing more scores on it.
	st, err := cricket.Play(cricket.Settings{Variant: cricket.VariantStandard}, []int64{1}, []throw.Dart{scored, dead})
	if err != nil {
		t.Fatal(err)
	}
	v := service.GameView{State: st}
	v.Game.Mode = game.ModeCricket
	v.Darts = []throw.Dart{scored}

	if hit, counted := targetResult(v, scored); !hit || !counted {
		t.Errorf("a dart with marks = %v, %v; want a counted hit", hit, counted)
	}
	if hit, counted := targetResult(v, dead); hit || !counted {
		t.Errorf("a dart without marks = %v, %v; want a counted miss", hit, counted)
	}
	if got := soundKind(v); got != "triple" {
		t.Errorf("sound of a triple that scored = %q, want triple", got)
	}
	v.Darts = []throw.Dart{scored, dead}
	if got := soundKind(v); got != "miss" {
		t.Errorf("sound of a dart without marks = %q, want miss", got)
	}
}

func TestLastTurnOfCricket(t *testing.T) {
	t.Parallel()
	st := cricket.State{Variant: cricket.VariantStandard, Turns: []cricket.Turn{
		{PlayerID: 1, Marks: []int{3, 1, 0}, Points: 40, Complete: true},
		{PlayerID: 2, Marks: []int{2, 0, 0}, Complete: true},
		{PlayerID: 1, Marks: []int{1}}, // still throwing
	}}
	v := service.GameView{State: st}
	texts := textsOf(i18n.English)

	if got, ok := lastTurnOf(texts, v, 1); !ok || got.Value != "4 marks · 40" {
		t.Errorf("player 1 = %+v, %v; want marks and points", got, ok)
	}
	if got, ok := lastTurnOf(texts, v, 2); !ok || got.Value != "2 marks" {
		t.Errorf("player 2 = %+v, %v; want only the marks", got, ok)
	}
	// In Cut-Throat the points went to the opponents.
	st.Variant = cricket.VariantCutThroat
	v = service.GameView{State: st}
	if got, ok := lastTurnOf(texts, v, 1); !ok || got.Value != "4 marks" {
		t.Errorf("cut-throat: player 1 = %+v, %v; want only the marks", got, ok)
	}
}

func TestBoardTargetsOfCricket(t *testing.T) {
	t.Parallel()
	// Both players close the 20, only player 1 the 19: the 20 is dead, the 19 still open.
	darts := []throw.Dart{
		{ID: 1, PlayerID: 1, Number: 20, Segment: dartboard.Triple},
		{ID: 2, PlayerID: 1, Number: 19, Segment: dartboard.Triple},
		{ID: 3, PlayerID: 1, Number: 0, Segment: dartboard.Miss},
		{ID: 4, PlayerID: 2, Number: 20, Segment: dartboard.Triple},
	}
	st, err := cricket.Play(cricket.Settings{Variant: cricket.VariantStandard}, []int64{1, 2}, darts)
	if err != nil {
		t.Fatal(err)
	}
	v := service.GameView{State: st}

	want := []uint8{19, 18, 17, 16, 15, dartboard.BullNumber}
	if got := boardTargetsOf(v, progressOf(v)); !slices.Equal(got, want) {
		t.Errorf("targets = %v, want %v", got, want)
	}
	if !isBoardTarget(want, "19") || isBoardTarget(want, "20") {
		t.Error("isBoardTarget doesn't match the numbers printed around the board")
	}
}
