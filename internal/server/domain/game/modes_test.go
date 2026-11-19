package game

import (
	"encoding/json"
	"testing"

	"github.com/sj14/darts-o-mat/internal/server/domain/game/aroundtheclock"
	"github.com/sj14/darts-o-mat/internal/server/domain/game/cricket"
	"github.com/sj14/darts-o-mat/internal/server/domain/game/x01"
)

func TestRulesMustFitTheMode(t *testing.T) {
	t.Parallel()
	ok := Settings{Mode: ModeX01, Legs: 1, Sets: 1, Rules: &x01.Settings{StartScore: 501, OutMode: x01.OutModeDouble}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, s := range map[string]Settings{
		"rules of another mode": {Mode: ModeX01, Legs: 1, Sets: 1, Rules: &aroundtheclock.Settings{}},
		"no rules":              {Mode: ModeX01, Legs: 1, Sets: 1},
		"unknown mode":          {Mode: "darts-bowling", Legs: 1, Sets: 1, Rules: &x01.Settings{StartScore: 501, OutMode: x01.OutModeDouble}},
		"invalid rules":         {Mode: ModeX01, Legs: 1, Sets: 1, Rules: &x01.Settings{}},
	} {
		if err := s.Validate(); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// The rules are written as "rules", and read into the type of the mode once it is prepared.
func TestRulesJSON(t *testing.T) {
	t.Parallel()
	out, err := json.Marshal(Settings{Mode: ModeAroundTheClock, Rules: &aroundtheclock.Settings{Bull: true}})
	if err != nil {
		t.Fatal(err)
	}
	var s Settings
	s.Rules = NewRules(ModeAroundTheClock)
	if err := json.Unmarshal(out, &s); err != nil {
		t.Fatal(err)
	}
	if r, ok := s.Rules.(*aroundtheclock.Settings); !ok || !r.Bull {
		t.Errorf("rules read back = %#v from %s", s.Rules, out)
	}
	if NewRules("nope") != nil {
		t.Error("rules of a mode that doesn't exist")
	}
}

// Cricket's variant is written by name, and an unknown one is refused.
func TestCricketRulesJSON(t *testing.T) {
	t.Parallel()
	ok := Settings{Mode: ModeCricket, Legs: 1, Sets: 1, Rules: &cricket.Settings{Variant: cricket.VariantCutThroat}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(ok)
	if err != nil {
		t.Fatal(err)
	}
	s := Settings{Rules: NewRules(ModeCricket)}
	if err := json.Unmarshal(out, &s); err != nil {
		t.Fatal(err)
	}
	if r, ok := s.Rules.(*cricket.Settings); !ok || r.Variant != cricket.VariantCutThroat {
		t.Errorf("rules read back = %#v from %s", s.Rules, out)
	}
	if err := (Settings{Mode: ModeCricket, Legs: 1, Sets: 1, Rules: &cricket.Settings{}}).Validate(); err == nil {
		t.Error("cricket without a variant is valid")
	}
	if err := (Settings{Mode: ModeCricket, Legs: 1, Sets: 1, Rules: &x01.Settings{StartScore: 501, OutMode: x01.OutModeDouble}}).Validate(); err == nil {
		t.Error("cricket with the rules of X01 is valid")
	}
}
