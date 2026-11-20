-- A rematch points to the game it is the rematch of; a game has at most one.
ALTER TABLE "games" ADD COLUMN "rematch_of" INTEGER;

--bun:split

CREATE UNIQUE INDEX "games_rematch_of" ON "games" ("rematch_of");
