-- Схема access — учётные данные входа и сеансы веба (барьер 1, AD-15, FR-128).
-- Журнал хранит факты (сотрудник заведён, учётная запись активирована, роль
-- назначена); секреты — только здесь: хеш пароля argon2id, счётчик неудач и
-- блокировка, сеансы scs (pgxstore). Выполняется ролью ant_owner; приложение
-- (ant_app) — полный доступ к схеме модуля.

-- +goose Up
CREATE SCHEMA access;
REVOKE ALL ON SCHEMA access FROM PUBLIC;

-- Учётные данные: заявка на регистрацию (pending) или учётная запись.
CREATE TABLE access.credentials (
    login        text        PRIMARY KEY CHECK (login ~ '^[a-z0-9._-]{3,64}$'),
    person_id    text        NOT NULL,
    hash         text        NOT NULL CHECK (hash LIKE '$argon2id$%'),
    status       text        NOT NULL CHECK (status IN ('pending', 'active', 'blocked')),
    display_name text        NOT NULL DEFAULT '',
    failures     integer     NOT NULL DEFAULT 0,
    locked_until timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX credentials_person ON access.credentials (person_id);

-- Сеансы веба scs (github.com/alexedwards/scs/pgxstore).
CREATE TABLE access.sessions (
    token  text        PRIMARY KEY,
    data   bytea       NOT NULL,
    expiry timestamptz NOT NULL
);
CREATE INDEX sessions_expiry_idx ON access.sessions (expiry);

GRANT USAGE ON SCHEMA access TO ant_app, ant_verifier;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA access TO ant_app;
-- Верификатору секреты не нужны: только учётные записи без хеша — через представление.
CREATE VIEW access.accounts AS SELECT login, person_id, status, created_at FROM access.credentials;
GRANT SELECT ON access.accounts TO ant_verifier;

-- +goose Down
DROP SCHEMA access CASCADE;
