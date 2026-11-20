package store

import (
	"context"
	"time"

	"github.com/sj14/darts-o-mat/internal/server/domain/account"
	"github.com/sj14/darts-o-mat/internal/server/domain/board"
	"github.com/sj14/darts-o-mat/internal/server/domain/game"
	"github.com/sj14/darts-o-mat/internal/server/domain/player"
	"github.com/sj14/darts-o-mat/internal/server/domain/site"
	"github.com/sj14/darts-o-mat/internal/server/domain/throw"
	"github.com/sj14/darts-o-mat/internal/server/domain/tournament"
)

// SiteStats counts what the server holds for the admin page, one query per table. The
// recent numbers count from now back site.RecentDays days, Active site.ActiveDays.
func (store *Store) SiteStats(ctx context.Context, now time.Time) (site.Stats, error) {
	recent := now.AddDate(0, 0, -site.RecentDays)
	active := now.AddDate(0, 0, -site.ActiveDays)
	var s site.Stats

	a := &s.Accounts
	err := store.db.NewSelect().Model((*account.Account)(nil)).
		ColumnExpr("COUNT(*)").
		ColumnExpr("COUNT(CASE WHEN last_seen_at > ? THEN 1 END)", active).
		ColumnExpr("COUNT(CASE WHEN last_seen_at > ? THEN 1 END)", recent).
		ColumnExpr("COUNT(CASE WHEN created_at > ? THEN 1 END)", recent).
		ColumnExpr("COUNT(CASE WHEN disabled THEN 1 END)").
		Scan(ctx, &a.Total, &a.Active, &a.ActiveRecent, &a.NewRecent, &a.Disabled)
	if err != nil {
		return s, err
	}

	err = store.db.NewSelect().Model((*player.Player)(nil)).
		ColumnExpr("COUNT(*)").Where("guest_of IS NOT NULL").Where("deleted_at IS NULL").
		Scan(ctx, &s.Guests)
	if err != nil {
		return s, err
	}

	g := &s.Games
	err = store.db.NewSelect().Model((*game.Game)(nil)).
		ColumnExpr("COUNT(CASE WHEN started_at IS NOT NULL AND completed_at IS NULL AND aborted_at IS NULL THEN 1 END)").
		ColumnExpr("COUNT(completed_at)").
		ColumnExpr("COUNT(aborted_at)").
		ColumnExpr("COUNT(CASE WHEN completed_at > ? THEN 1 END)", recent).
		Scan(ctx, &g.Running, &g.Completed, &g.Aborted, &g.CompletedRecent)
	if err != nil {
		return s, err
	}
	var byMode []struct {
		Mode  game.Mode
		Games int
	}
	err = store.db.NewSelect().Model((*game.Game)(nil)).
		Column("mode").ColumnExpr("COUNT(*) AS games").
		Where("completed_at IS NOT NULL").Group("mode").
		Scan(ctx, &byMode)
	if err != nil {
		return s, err
	}
	g.CompletedByMode = map[game.Mode]int{}
	for _, m := range byMode {
		g.CompletedByMode[m.Mode] = m.Games
	}

	t := &s.Tournaments
	err = store.db.NewSelect().Model((*tournament.Tournament)(nil)).
		ColumnExpr("COUNT(CASE WHEN started_at IS NOT NULL AND completed_at IS NULL AND aborted_at IS NULL THEN 1 END)").
		ColumnExpr("COUNT(completed_at)").
		Scan(ctx, &t.Running, &t.Completed)
	if err != nil {
		return s, err
	}

	d := &s.Darts
	err = store.db.NewSelect().Model((*throw.Dart)(nil)).
		ColumnExpr("COUNT(*)").
		ColumnExpr("COUNT(x)").
		ColumnExpr("COUNT(CASE WHEN created_at > ? THEN 1 END)", recent).
		Where("NOT takeout").Where("NOT leave").Where("NOT skip").
		Scan(ctx, &d.Total, &d.Camera, &d.Recent)
	if err != nil {
		return s, err
	}

	b := &s.Boards
	err = store.db.NewSelect().Model((*board.Board)(nil)).
		ColumnExpr("COUNT(*)").
		ColumnExpr("COUNT(CASE WHEN id IN (SELECT board_id FROM board_tokens) THEN 1 END)").
		ColumnExpr("COUNT(CASE WHEN last_used_at > ? THEN 1 END)", recent).
		Scan(ctx, &b.Total, &b.Camera, &b.UsedRecent)
	return s, err
}
