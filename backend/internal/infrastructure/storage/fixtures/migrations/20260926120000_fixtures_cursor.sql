-- +goose Up
-- Курсор мира заготовок (AD-36): положение сценария, общее для копий api.
CREATE SCHEMA IF NOT EXISTS fixtures;
CREATE TABLE IF NOT EXISTS fixtures.cursor (
    name       text PRIMARY KEY DEFAULT 'default',
    scenario   text        NOT NULL,
    run_id     text        NOT NULL DEFAULT '',
    step       integer     NOT NULL CHECK (step >= 0),
    paused     boolean     NOT NULL DEFAULT true,
    speed      integer     NOT NULL DEFAULT 1 CHECK (speed BETWEEN 1 AND 1000),
    clock_at   timestamptz NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS fixtures.cursor;
DROP SCHEMA IF EXISTS fixtures;
