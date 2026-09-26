-- +goose Up
-- Схема движка (эпик 07, AD-1, AD-45): проекции каркаса (один писатель на
-- проекцию, запись — только эффектами в транзакции journal.Append), вклады
-- изделий в показатели (заменяются целиком при пересвёртке) и журнал
-- изменений сущностей для живых обновлений SSE (AD-6, AD-21).
CREATE SCHEMA IF NOT EXISTS engine;

CREATE TABLE engine.projections (
    name    text  NOT NULL,
    key     text  NOT NULL,
    item_id text  NOT NULL DEFAULT '',
    value   jsonb NOT NULL,
    PRIMARY KEY (name, key)
);
CREATE INDEX projections_item ON engine.projections (name, item_id);

CREATE TABLE engine.contributions (
    item_id text     NOT NULL,
    metric  text     NOT NULL,
    slice   text     NOT NULL,
    value   bigint   NOT NULL,
    scale   integer  NOT NULL DEFAULT 0,
    sources text[]   NOT NULL DEFAULT '{}'
);
CREATE INDEX contributions_item ON engine.contributions (item_id);
CREATE INDEX contributions_metric ON engine.contributions (metric, slice);

-- Журнал изменений: строка на сущность; seq — позиция журнала, после которой
-- сущность изменилась (id события SSE). NOTIFY ant_changes несёт только seq.
CREATE TABLE engine.changes (
    n           bigserial   PRIMARY KEY,
    seq         bigint      NOT NULL,
    entity      text        NOT NULL,
    id          text        NOT NULL,
    run_id      text        NOT NULL DEFAULT '',
    received_at timestamptz
);
CREATE INDEX changes_seq ON engine.changes (seq);

-- +goose Down
DROP SCHEMA engine CASCADE;
