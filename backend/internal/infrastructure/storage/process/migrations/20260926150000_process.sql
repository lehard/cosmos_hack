-- Схема process — версии процесса BPMN (AD-17, FR-22, FR-23): подписываемые
-- байты XML как загружены, хеш при загрузке, статус и подписи кворума.
-- Журнал остаётся источником истины о запуске изделий: хеш закреплённой
-- версии записан в item.item.registered; изменение xml здесь в обход
-- системы обнаруживается сравнением H(xml) с этим хешем (версия не
-- исполняется). Проекции модуля — в engine.projections (писатель process).
-- Выполняется ролью ant_owner; приложение (ant_app) — полный доступ к схеме модуля.

-- +goose Up
CREATE SCHEMA process;
REVOKE ALL ON SCHEMA process FROM PUBLIC;

CREATE TABLE process.versions (
    version_id            text        PRIMARY KEY,
    label                 text        NOT NULL,
    status                text        NOT NULL CHECK (status IN ('draft', 'on_approval', 'active', 'retired')),
    base_version_id       text        NOT NULL DEFAULT '',
    author                text        NOT NULL DEFAULT '',
    hash                  text        NOT NULL,
    xml                   bytea       NOT NULL,
    created_at            timestamptz NOT NULL,
    effective_from        timestamptz,
    genesis               boolean     NOT NULL DEFAULT false,
    signatures            jsonb       NOT NULL DEFAULT '[]',
    route_closed_event_id text        NOT NULL DEFAULT ''
);
CREATE INDEX versions_hash ON process.versions (hash);

GRANT USAGE ON SCHEMA process TO ant_app, ant_verifier;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA process TO ant_app;
GRANT SELECT ON ALL TABLES IN SCHEMA process TO ant_verifier;

-- +goose Down
DROP SCHEMA process CASCADE;
