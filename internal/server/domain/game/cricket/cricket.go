// Package cricket is the game mode Cricket: the players close the numbers 15 to 20 and
// the bull by hitting each of them three times; further hits on a number score while an
// opponent hasn't closed it yet.
package cricket

import (
	"errors"
	"fmt"

	"github.com/sj14/darts-o-mat/internal/common/dartboard"
	"github.com/sj14/darts-o-mat/internal/server/domain/game/turn"
	"github.com/sj14/darts-o-mat/internal/server/domain/throw"
)

// Targets are the numbers of the game, in the order a scoreboard shows them.
var Targets = [NumTargets]uint8{20, 19, 18, 17, 16, 15, dartboard.BullNumber}

// NumTargets is how many numbers a player has to close.
const NumTargets = 7

// MarksToClose is how many marks close a number: a single counts one, a double two and
// a triple three; the bull one and the bullseye two.
const MarksToClose = 3

// Variant is who gets the points of a number that is still open.
type Variant uint8

const (
	_                Variant = iota
	VariantStandard          // the thrower scores, the most points win
	VariantCutThroat         // the opponents who haven't closed the number get them, the fewest points win
)

var variantNames = map[Variant]string{
	VariantStandard:  "standard",
	VariantCutThroat: "cut-throat",
}

func (v Variant) MarshalText() ([]byte, error) {
	name, ok := variantNames[v]
	if !ok {
		return nil, fmt.Errorf("unknown variant %d", v)
	}
	return []byte(name), nil
}

func (v *Variant) UnmarshalText(text []byte) error {
	for variant, name := range variantNames {
		if name == string(text) {
			*v = variant
			return nil
		}
	}
	return fmt.Errorf("unknown variant %q, use standard or cut-throat", text)
}

type Settings struct {
	Variant Variant `json:"variant"`
}

func (s Settings) Validate() error {
	if _, ok := variantNames[s.Variant]; !ok {
		return errors.New("variant must be standard or cut-throat")
	}
	return nil
}

type PlayerState struct {
	PlayerID int64 `json:"player_id"`
	// Marks per number of Targets in the current leg, 0 to MarksToClose.
	Marks  [NumTargets]uint8 `json:"marks"`
	Points int               `json:"points"` // in the current leg
	turn.Record
	Stats Stats `json:"stats"`
}

// Closed reports whether the player has closed every number.
func (p PlayerState) Closed() bool {
	for _, m := range p.Marks {
		if m < MarksToClose {
			return false
		}
	}
	return true
}

// Stats of a player in one game, over all legs.
type Stats struct {
	Darts int `json:"darts"`
	// Marks counts the marks that closed a number or scored; marks on a number everybody
	// has closed count nothing.
	Marks int     `json:"marks"`
	MPR   float64 `json:"mpr"` // marks per round of three darts
}

type State struct {
	turn.State
	Variant Variant       `json:"variant"`
	Players []PlayerState `json:"players"`
	// Turns lists every turn in the order they were thrown; the last one may still be in progress.
	Turns []Turn `json:"turns"`
}

// Turn is one player's turn of up to three darts.
type Turn struct {
	PlayerID int64        `json:"player_id"`
	Set      int          `json:"set"`
	Leg      int          `json:"leg"` // in the set
	Round    int          `json:"round"`
	Darts    []throw.Dart `json:"darts"`
	Marks    []int        `json:"marks"`    // per dart: the marks that counted, see Stats.Marks
	Points   int          `json:"points"`   // scored in this turn, by the thrower or (cut-throat) for the opponents
	Complete bool         `json:"complete"` // false while the player still throws
}

// Play replays the darts of a game and returns the resulting state.
// A player who has closed every number and is ahead on points (Standard), or not behind
// (Cut-Throat), wins the leg, see package turn.
func Play(s Settings, playerIDs []int64, darts []throw.Dart) (State, error) {
	return PlayLog(s, playerIDs, turn.Log{Entries: darts})
}

// Player returns the state of a player of the game.
func (s State) Player(id int64) (PlayerState, bool) {
	for _, p := range s.Players {
		if p.PlayerID == id {
			return p, true
		}
	}
	return PlayerState{}, false
}

// Dead reports whether the number Targets[k] is dead: every player still in the game has
// closed it, so more marks on it score nothing.
func (s State) Dead(k int) bool {
	for _, p := range s.Players {
		if !p.Left && p.Marks[k] < MarksToClose {
			return false
		}
	}
	return true
}

// PlayLog replays a game's log of darts and takeouts, see Play.
func PlayLog(s Settings, playerIDs []int64, log turn.Log) (State, error) {
	st := State{Variant: s.Variant, Turns: []Turn{}}
	for _, id := range playerIDs {
		st.Players = append(st.Players, PlayerState{PlayerID: id})
	}

	t := log.Turn(len(st.Players))
	t.OnLegStart = func() {
		for i := range st.Players {
			st.Players[i].Marks, st.Players[i].Points = [NumTargets]uint8{}, 0
		}
	}
	var cur *Turn // the turn being thrown

	err := log.Replay(t, func(i int, d throw.Dart) error {
		if err := d.Validate(); err != nil {
			return fmt.Errorf("dart %d: %w", i+1, err)
		}

		p := &st.Players[t.Player]
		if t.Dart == 0 {
			st.Turns = append(st.Turns, Turn{PlayerID: p.PlayerID, Set: t.Set, Leg: t.Leg, Round: t.Round})
			cur = &st.Turns[len(st.Turns)-1]
		}
		marks, points := st.throw(s.Variant, t, d)
		p.Stats.Darts++
		p.Stats.Marks += marks
		cur.Darts, cur.Marks = append(cur.Darts, d), append(cur.Marks, marks)
		cur.Points += points

		if st.wins(s.Variant, t) {
			cur.Complete = true
			t.Finish()
			return nil
		}
		t.Advance()
		cur.Complete = t.Dart == 0 || t.Waiting()
		return nil
	})
	if err != nil {
		return State{}, err
	}

	for i := range st.Players {
		p := &st.Players[i]
		p.Record = t.Records[i]
		if p.Stats.Darts > 0 {
			p.Stats.MPR = float64(p.Stats.Marks) * turn.DartsPerTurn / float64(p.Stats.Darts)
		}
	}

	st.State = t.State(playerIDs)
	return st, nil
}

// throw counts dart d of the player who is up: it closes their number, and what is left
// of its marks scores while an opponent still has the number open. It returns the marks
// that counted and the points scored.
func (st *State) throw(v Variant, t *turn.Turn, d throw.Dart) (marks, points int) {
	k := targetIndex(d)
	if k < 0 {
		return 0, 0
	}
	p := &st.Players[t.Player]
	hits := uint8(d.Segment.Multiplier())
	closing := min(hits, MarksToClose-p.Marks[k])
	p.Marks[k] += closing
	marks = int(closing)
	extra := int(hits - closing)
	if extra == 0 {
		return marks, 0
	}

	value := int(Targets[k]) * extra
	open := 0 // opponents still in the game who haven't closed the number
	for i := range st.Players {
		o := &st.Players[i]
		if i == t.Player || t.Records[i].Left || o.Marks[k] >= MarksToClose {
			continue
		}
		open++
		if v == VariantCutThroat {
			o.Points += value
		}
	}
	if open == 0 {
		return marks, 0
	}
	if v == VariantCutThroat {
		return marks + extra, value * open
	}
	p.Points += value
	return marks + extra, value
}

// wins reports whether the player who is up wins the leg: they have closed every number
// and no opponent still in the game is ahead of them, i.e. has more points (Standard) or
// fewer (Cut-Throat). Equal points are enough.
func (st *State) wins(v Variant, t *turn.Turn) bool {
	p := st.Players[t.Player]
	if !p.Closed() {
		return false
	}
	for i, o := range st.Players {
		if i == t.Player || t.Records[i].Left {
			continue
		}
		if v == VariantCutThroat && o.Points < p.Points || v == VariantStandard && o.Points > p.Points {
			return false
		}
	}
	return true
}

// targetIndex returns the index in Targets of the number the dart hit, -1 for a miss or
// a number that isn't part of the game.
func targetIndex(d throw.Dart) int {
	if d.Segment == dartboard.Miss {
		return -1
	}
	for k, n := range Targets {
		if n == d.Number {
			return k
		}
	}
	return -1
}
