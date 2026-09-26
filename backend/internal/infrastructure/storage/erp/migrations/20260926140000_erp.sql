-- Схема erp — изменяемое операционное состояние модуля erp вне доказательств
-- (AD-1, AD-7, AD-18): очередь исходящих роли outbox, состояние каналов обмена
-- и шлюз входящих. Журнал остаётся единственным источником истины: запрос,
-- ответ, карантин и решения людей — записи erp.posting.*; очередь — их
-- проекция (колонки view/token/status пишет потребитель erp.outbox в
-- транзакции курсора), попытки — транспортное состояние отправителя.
-- Выполняется ролью ant_owner; приложение (ant_app) — полный доступ к схеме модуля.

-- +goose Up
CREATE SCHEMA erp;
REVOKE ALL ON SCHEMA erp FROM PUBLIC;

-- Очередь исходящих (FR-96): одно сообщение — один бизнес-ключ (AD-7).
CREATE TABLE erp.outbox (
    business_key text        PRIMARY KEY,
    system       text        NOT NULL,
    status       text        NOT NULL,
    token        text        NOT NULL,
    item_id      text        NOT NULL DEFAULT '',
    first_seq    bigint      NOT NULL DEFAULT 0,
    view         jsonb       NOT NULL,
    handled      text        NOT NULL DEFAULT '',
    retry_token  text        NOT NULL DEFAULT '',
    tries        integer     NOT NULL DEFAULT 0,
    next_at      timestamptz NOT NULL DEFAULT 'epoch',
    last_error   text        NOT NULL DEFAULT '',
    updated      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX outbox_due ON erp.outbox (system, status, next_at);

-- Каналы обмена: сверка ответной стороны при старте и по расписанию (AD-18).
CREATE TABLE erp.channels (
    system           text        PRIMARY KEY,
    state            text        NOT NULL CHECK (state IN ('ok', 'degraded', 'disabled')),
    endpoint         text        NOT NULL DEFAULT '',
    stand            boolean     NOT NULL DEFAULT false,
    contract_version text        NOT NULL DEFAULT '',
    detail           text        NOT NULL DEFAULT '',
    checked_at       timestamptz NOT NULL,
    last_exchange_at timestamptz
);

-- Шлюз входящих: поданные в приём факты и их source_seq (AD-7: повторная
-- подача того же факта — с тем же номером), следующий source_seq источника.
CREATE TABLE erp.gateway_seen (
    source_id  text   NOT NULL,
    event_id   text   NOT NULL,
    source_seq bigint NOT NULL,
    PRIMARY KEY (source_id, event_id)
);
CREATE TABLE erp.gateway_seq (
    source_id text   PRIMARY KEY,
    next_seq  bigint NOT NULL
);

GRANT USAGE ON SCHEMA erp TO ant_app, ant_verifier;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA erp TO ant_app;
GRANT SELECT ON ALL TABLES IN SCHEMA erp TO ant_verifier;

-- +goose Down
DROP SCHEMA erp CASCADE;
