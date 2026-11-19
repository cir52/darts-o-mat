-- Cricket keeps its settings in a table of its own, like every game mode.
CREATE TABLE "game_settings_cricket" ("id" BIGSERIAL NOT NULL, "variant" SMALLINT, PRIMARY KEY ("id"));
