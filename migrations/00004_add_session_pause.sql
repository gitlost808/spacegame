-- +goose Up
ALTER TABLE game_sessions
    ADD COLUMN paused boolean NOT NULL DEFAULT false,
    ADD COLUMN paused_us bigint NOT NULL DEFAULT 0 CHECK (paused_us >= 0),
    ADD COLUMN revision bigint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE game_sessions
    DROP COLUMN revision,
    DROP COLUMN paused_us,
    DROP COLUMN paused;
