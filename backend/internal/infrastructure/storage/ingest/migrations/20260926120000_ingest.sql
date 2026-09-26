-- Схема ingest — изменяемое состояние приёма вне доказательств (AD-1, AD-7, FR-30):
-- реестр идемпотентности, учёт source_seq по источникам и карантин сообщений.
-- Журнал остаётся единственным источником истины: факт помещения в карантин —
-- служебная запись ingest.message.quarantined, содержимое — в хранилище
-- материалов по адресу H(байты) (AD-2). Строки реестра и карантина пишутся в
-- транзакции journal.Append (AppendRequest.Project, AD-45), поэтому
-- расхождение «факт в журнале — ключа нет в реестре» невозможно.
-- Выполняется ролью ant_owner; приложение (ant_app) — полный доступ к схеме модуля.

-- +goose Up
CREATE SCHEMA ingest;
REVOKE ALL ON SCHEMA ingest FROM PUBLIC;

-- Ключи идемпотентности: source_id + event_id → отпечаток канонического payload (AD-7, FR-31).
CREATE TABLE ingest.seen (
    source_id          text    NOT NULL,
    event_id           text    NOT NULL,
    fingerprint        text    NOT NULL,
    seq                bigint  NOT NULL DEFAULT 0,
    source_seq         bigint  NOT NULL DEFAULT 0,
    signature_verified boolean NOT NULL,
    PRIMARY KEY (source_id, event_id)
);

-- Учёт source_seq источника: наибольший номер, разрывы, вид, ключ, часы (FR-33, FR-39).
CREATE TABLE ingest.sources (
    source_id text  PRIMARY KEY,
    state     jsonb NOT NULL
);

-- Карантин сообщений (FR-30): повтор тех же байтов с тем же кодом — та же запись.
CREATE TABLE ingest.quarantine (
    id               uuid        PRIMARY KEY,
    journal_seq      bigint      NOT NULL DEFAULT 0,
    source_id        text        NOT NULL,
    event_id         text        NOT NULL DEFAULT '',
    source_seq       bigint      NOT NULL DEFAULT 0,
    event_type       text        NOT NULL DEFAULT '',
    fingerprint      text        NOT NULL,
    material_address text        NOT NULL,
    code             text        NOT NULL,
    field            text        NOT NULL DEFAULT '',
    detail           text        NOT NULL DEFAULT '',
    status           text        NOT NULL CHECK (status IN ('open', 'accepted', 'still_invalid', 'discarded')),
    received_at      timestamptz NOT NULL,
    raw              bytea,
    resolved_by      text        NOT NULL DEFAULT '',
    created          bigserial,
    UNIQUE (fingerprint, code)
);
CREATE INDEX quarantine_status ON ingest.quarantine (status, created DESC);
CREATE INDEX quarantine_source ON ingest.quarantine (source_id, created DESC);

GRANT USAGE ON SCHEMA ingest TO ant_app, ant_verifier;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA ingest TO ant_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA ingest TO ant_app;
GRANT SELECT ON ALL TABLES IN SCHEMA ingest TO ant_verifier;

-- +goose Down
DROP SCHEMA ingest CASCADE;
