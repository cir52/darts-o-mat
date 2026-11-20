package service

import (
	"context"
	"errors"
	"testing"

	"github.com/sj14/darts-o-mat/internal/server/domain/account"
	"github.com/sj14/darts-o-mat/internal/server/domain/game"
	"github.com/sj14/darts-o-mat/internal/server/domain/game/x01"
)

// A rematch has the old game's settings; the starter's players sit in it on their boards,
// the other players with an account are invited even without being friends, and whoever
// asks for it as well gets the same one.
func TestRematch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newService(t)
	anna, ben, carla := register(t, s, "anna"), register(t, s, "ben"), register(t, s, "carla")
	keller, err := s.CreateBoard(ctx, anna, "Keller")
	if err != nil {
		t.Fatal(err)
	}
	guest, err := s.CreateGuest(ctx, anna, "Gast")
	if err != nil {
		t.Fatal(err)
	}
	// ben joins an open game: anna and ben aren't friends.
	g := quick(anna.PlayerID)
	g.Participants[0].BoardID = &keller.ID
	g.Seats, g.TurnSeconds = 3, 60
	v, err := createGame(ctx, s, anna, g, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := v.Game.ID
	if err := s.AddPlayer(ctx, anna, GameLobby(id), guest.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := joinGame(ctx, s, ben, id, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Rematch(ctx, anna, id); !errors.Is(err, ErrNoRematch) {
		t.Errorf("rematch of a running game: got %v, want ErrNoRematch", err)
	}
	finish(t, s, id, anna, ben)
	if _, err := s.Rematch(ctx, carla, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("a stranger asks for a rematch: got %v, want ErrNotFound", err)
	}

	r, err := s.Rematch(ctx, anna, id)
	if err != nil {
		t.Fatal(err)
	}
	if r.Started || r.Game.Open() || r.Game.TurnSeconds != 60 || r.Game.Rules.(*x01.Settings).StartScore != 40 || r.Game.RematchOf == nil || *r.Game.RematchOf != id {
		t.Errorf("rematch = %+v, want a lobby with the old settings, not open", r.Game)
	}
	want := map[int64]game.InvitationStatus{anna.PlayerID: game.InvitationAccepted, guest.ID: game.InvitationAccepted, ben.PlayerID: game.InvitationPending}
	if len(r.Game.Participants) != len(want) {
		t.Errorf("participants = %+v, want %v", r.Game.Participants, want)
	}
	for _, p := range r.Game.Participants {
		if want[p.PlayerID] != p.Status {
			t.Errorf("player %d: status %q, want %q", p.PlayerID, p.Status, want[p.PlayerID])
		}
	}
	if b := r.BoardOf(anna.PlayerID); b == nil || *b != keller.ID {
		t.Errorf("anna's board = %v, want Keller", b)
	}
	if old, _ := s.Game(ctx, anna, id); old.RematchID != r.Game.ID || old.CanRematch() || !old.Completed {
		t.Errorf("old game: rematch %d, can rematch %v, completed %v; want %d, false, true", old.RematchID, old.CanRematch(), old.Completed, r.Game.ID)
	}

	again, err := s.Rematch(ctx, ben, id)
	if err != nil || again.Game.ID != r.Game.ID || !again.Invited {
		t.Fatalf("ben asks for a rematch too: got game %d (invited %v), %v; want %d", again.Game.ID, again.Invited, err, r.Game.ID)
	}
	if _, err := acceptGame(ctx, s, ben, r.Game.ID, nil); err != nil {
		t.Fatal(err)
	}
	if r, _ = s.Game(ctx, anna, r.Game.ID); !r.Started {
		t.Error("the rematch didn't start once ben accepted")
	}
}

// Without anyone to invite, e.g. against a guest, the rematch starts at once. Who left the
// game isn't in it, and who left can't ask for one.
func TestRematchStartsAtOnce(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newService(t)
	anna, ben := register(t, s, "anna"), register(t, s, "ben")
	befriend(t, s, anna, ben)
	guest, err := s.CreateGuest(ctx, anna, "Gast")
	if err != nil {
		t.Fatal(err)
	}
	v, err := createGame(ctx, s, anna, quick(anna.PlayerID, guest.ID), []string{"ben"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acceptGame(ctx, s, ben, v.Game.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LeaveGame(ctx, ben, v.Game.ID); err != nil {
		t.Fatal(err)
	}
	finish(t, s, v.Game.ID, anna)

	if _, err := s.Rematch(ctx, ben, v.Game.ID); !errors.Is(err, ErrNoRematch) {
		t.Errorf("ben left and asks for a rematch: got %v, want ErrNoRematch", err)
	}
	r, err := s.Rematch(ctx, anna, v.Game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Started || len(r.Game.Participants) != 2 {
		t.Errorf("rematch started %v with %+v, want anna and her guest at once", r.Started, r.Game.Participants)
	}
	if old, _ := s.Game(ctx, ben, v.Game.ID); old.RematchID != 0 || old.CanRematch() {
		t.Errorf("ben sees rematch %d (can rematch %v), want none: he isn't in it", old.RematchID, old.CanRematch())
	}
}

// The matches of a tournament have no rematch.
func TestRematchNotInTournament(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newService(t)
	anna := register(t, s, "anna")
	guest, err := s.CreateGuest(ctx, anna, "Gast")
	if err != nil {
		t.Fatal(err)
	}
	v, err := createGame(ctx, s, anna, quick(anna.PlayerID, guest.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	finish(t, s, v.Game.ID, anna)
	if v, err = s.Game(ctx, anna, v.Game.ID); err != nil || !v.CanRematch() {
		t.Fatalf("a single game: can rematch %v, %v; want true", v.CanRematch(), err)
	}
	tournamentID := int64(1)
	v.Game.TournamentID = &tournamentID
	if v.CanRematch() {
		t.Error("a tournament's match can have a rematch")
	}
}

// finish ends a game of quick with one dart of the player who is up, one of the accounts'.
func finish(t *testing.T, s *Service, id int64, accounts ...account.Account) {
	t.Helper()
	for _, acc := range accounts {
		if _, err := s.AddDart(context.Background(), acc, id, d20); err == nil {
			return
		}
	}
	t.Fatal("nobody could throw the last dart")
}
