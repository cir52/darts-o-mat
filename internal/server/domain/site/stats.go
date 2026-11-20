package site

import "github.com/sj14/darts-o-mat/internal/server/domain/game"

// Stats are numbers about the whole server for the admin page, counted from the stored
// data. "Recent" is the last RecentDays days.
type Stats struct {
	Accounts    AccountStats
	Guests      int // guests of accounts, without deleted ones
	Games       GameStats
	Tournaments TournamentStats
	Darts       DartStats
	Boards      BoardStats
}

// RecentDays is how far back the recent numbers of Stats count.
const RecentDays = 30

// ActiveDays is the shorter period for the accounts active lately.
const ActiveDays = 7

// AccountStats counts the accounts.
type AccountStats struct {
	Total        int
	Active       int // with a request in the last ActiveDays days
	ActiveRecent int // with a request in the last RecentDays days
	NewRecent    int // registered in the last RecentDays days
	Disabled     int
}

// GameStats counts the games; open lobbies count in none of them.
type GameStats struct {
	Running         int
	Completed       int
	Aborted         int
	CompletedRecent int // completed in the last RecentDays days
	// CompletedByMode is the number of completed games per mode; a mode without one is
	// missing.
	CompletedByMode map[game.Mode]int
}

// TournamentStats counts the tournaments; those that haven't started count in neither.
type TournamentStats struct {
	Running   int
	Completed int
}

// DartStats counts the darts thrown in games, without takeouts and other log entries.
type DartStats struct {
	Total  int
	Camera int // detected by a camera, i.e. with a position
	Recent int // thrown in the last RecentDays days
}

// BoardStats counts the boards of the accounts.
type BoardStats struct {
	Total      int
	Camera     int // with a camera token
	UsedRecent int // a game started on it in the last RecentDays days
}
