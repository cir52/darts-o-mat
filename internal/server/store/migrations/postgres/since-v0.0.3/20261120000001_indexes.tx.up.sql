-- Replaying a game reads its darts (store.ListDarts, DeleteLastDart); the darts of recent
-- games (store.dartSince, the admin's statistics) and the heatmap find them through the games.
CREATE INDEX "darts_game_id" ON "darts" ("game_id");

--bun:split

-- The games played on a board, on every live update (store.ListUnfinishedGamesByBoard).
CREATE INDEX "game_players_board_id" ON "game_players" ("board_id");

--bun:split

-- The games of a player: game lists, invitations, presence, and the heatmap, which finds
-- the player's darts through their games (store.ListPositions).
CREATE INDEX "game_players_player_id" ON "game_players" ("player_id");

--bun:split

-- The results of a player: the form in the open lobbies and the statistics page.
CREATE INDEX "game_stats_player_id" ON "game_stats" ("player_id");

--bun:split

-- The thumbs of a player, each rater's latest (store.Reputations).
CREATE INDEX "ratings_rated_id_rater_id_game_id" ON "ratings" ("rated_id", "rater_id", "game_id");
