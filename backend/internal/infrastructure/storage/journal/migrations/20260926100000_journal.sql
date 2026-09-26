-- Журнал — единственный источник истины (AD-2, AD-44, FR-71).
-- Выполняется ролью ant_owner (ant migrate делает SET ROLE ant_owner): всеми
-- объектами владеет ant_owner, приложение (ant_app) только дописывает и читает,
-- верификатор (ant_verifier) только читает (AD-1).
--
-- Схема journal — неизменяемая часть: записи обеих цепочек и леджер лимитов
-- разрешений на отклонение. Изменение и удаление невозможны: у ant_app нет
-- UPDATE/DELETE/TRUNCATE, строчный триггер BEFORE UPDATE OR DELETE и триггер
-- оператора BEFORE TRUNCATE отвергают их у любого, кроме суперпользователя с
-- session_replication_role = replica — такой обход обнаруживает хранитель (AD-8).
--
-- Схема journal_state — изменяемое служебное состояние модуля journal вне
-- доказательств: аренды с эпохами (AD-6) и курсоры потребителей (AD-45).
-- Вынесена отдельно, чтобы в схеме journal у ant_app было только INSERT и SELECT.

-- +goose Up
CREATE SCHEMA journal;
CREATE SCHEMA journal_state;
REVOKE ALL ON SCHEMA journal, journal_state FROM PUBLIC;

-- Записи обеих цепочек (main и ca). seq — позиция в своей цепочке (AD-37).
-- header — JCS открытых полей без link (entry.schema.json): хранится как есть,
-- поэтому верификатор пересчитывает звено по тем же байтам; столбцы рядом —
-- только для поиска и проверок AD-39.
CREATE TABLE journal.entries (
    chain          text        NOT NULL CHECK (chain IN ('main', 'ca')),
    seq            bigint      NOT NULL CHECK (seq >= 1),
    event_id       uuid        NOT NULL,
    event_type     text        NOT NULL,
    entry_kind     text        NOT NULL CHECK (entry_kind IN ('fact', 'reaction', 'decision', 'service')),
    stream         text        NOT NULL,
    partition      integer     NOT NULL CHECK (partition >= 0),
    item_id        text,
    run_id         text,
    source_id      text        NOT NULL,
    occurred_at    timestamptz NOT NULL,
    recorded_at    timestamptz NOT NULL,
    committed_at   timestamptz NOT NULL,
    -- Пометки каталога (AD-40): версия потока для гардов и триггер свёртки (AD-5).
    guard_relevant boolean     NOT NULL,
    is_trigger     boolean     NOT NULL,
    commit         bytea       NOT NULL CHECK (length(commit) = 32),
    link           bytea       NOT NULL CHECK (length(link) = 32),
    header         text        NOT NULL,
    -- TODO(29): salt и конверт — внутри зашифрованного блока (AD-23, DEK/KEK);
    -- в демо-треке хранятся открыто, соль пустая.
    salt           bytea       NOT NULL,
    envelope       bytea       NOT NULL,
    PRIMARY KEY (chain, seq),
    UNIQUE (chain, event_id)
);

CREATE INDEX entries_stream    ON journal.entries (stream, seq)       WHERE chain = 'main';
CREATE INDEX entries_partition ON journal.entries (partition, seq)    WHERE chain = 'main';
CREATE INDEX entries_trigger   ON journal.entries (partition, seq)    WHERE chain = 'main' AND is_trigger;
CREATE INDEX entries_recorded  ON journal.entries (recorded_at, seq)  WHERE chain = 'main';
CREATE INDEX entries_occurred  ON journal.entries (occurred_at, seq)  WHERE chain = 'main';
CREATE INDEX entries_type      ON journal.entries (event_type, seq)   WHERE chain = 'main';
CREATE INDEX entries_run       ON journal.entries (run_id, seq)       WHERE chain = 'main' AND run_id IS NOT NULL;

-- Леджер лимитов разрешений на отклонение (AD-39, FR-54): открытие лимита — +N,
-- расход — −N атомарно с решением; остаток — сумма. Только дописывание.
CREATE TABLE journal.concession_ledger (
    concession_id text   NOT NULL,
    seq           bigint NOT NULL,
    delta         bigint NOT NULL CHECK (delta <> 0)
);
CREATE INDEX concession_ledger_id ON journal.concession_ledger (concession_id);

-- +goose StatementBegin
CREATE FUNCTION journal.forbid_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'journal.append_only: % в %.% запрещено — журнал только на дописывание (FR-71, AD-2)',
        TG_OP, TG_TABLE_SCHEMA, TG_TABLE_NAME
        USING ERRCODE = 'AJ001';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER entries_forbid_change BEFORE UPDATE OR DELETE ON journal.entries
    FOR EACH ROW EXECUTE FUNCTION journal.forbid_change();
CREATE TRIGGER entries_forbid_truncate BEFORE TRUNCATE ON journal.entries
    FOR EACH STATEMENT EXECUTE FUNCTION journal.forbid_change();
CREATE TRIGGER ledger_forbid_change BEFORE UPDATE OR DELETE ON journal.concession_ledger
    FOR EACH ROW EXECUTE FUNCTION journal.forbid_change();
CREATE TRIGGER ledger_forbid_truncate BEFORE TRUNCATE ON journal.concession_ledger
    FOR EACH STATEMENT EXECUTE FUNCTION journal.forbid_change();
-- Триггеры срабатывают и при session_replication_role = replica (ALWAYS):
-- выключить их может только владелец таблицы или суперпользователь.
ALTER TABLE journal.entries ENABLE ALWAYS TRIGGER entries_forbid_change;
ALTER TABLE journal.entries ENABLE ALWAYS TRIGGER entries_forbid_truncate;
ALTER TABLE journal.concession_ledger ENABLE ALWAYS TRIGGER ledger_forbid_change;
ALTER TABLE journal.concession_ledger ENABLE ALWAYS TRIGGER ledger_forbid_truncate;

-- Аренды партиций и ролей-лидеров с эпохой (AD-6). Время — только InfraClock (AD-37).
CREATE TABLE journal_state.leases (
    name        text        PRIMARY KEY,
    holder      text        NOT NULL,
    epoch       bigint      NOT NULL CHECK (epoch >= 1),
    expires_at  timestamptz NOT NULL,
    acquired_at timestamptz NOT NULL
);

-- Курсоры потребителей журнала (AD-45): partition = -1 у глобальных.
CREATE TABLE journal_state.consumer_offsets (
    name       text        NOT NULL,
    partition  integer     NOT NULL CHECK (partition >= -1),
    seq        bigint      NOT NULL CHECK (seq >= 0),
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (name, partition)
);

GRANT USAGE ON SCHEMA journal, journal_state TO ant_app, ant_verifier;
GRANT SELECT, INSERT ON journal.entries, journal.concession_ledger TO ant_app;
GRANT SELECT ON journal.entries, journal.concession_ledger TO ant_verifier;
GRANT SELECT, INSERT, UPDATE, DELETE ON journal_state.leases, journal_state.consumer_offsets TO ant_app;
GRANT SELECT ON journal_state.leases, journal_state.consumer_offsets TO ant_verifier;
-- Откат (Down) не предусмотрен: журнал не удаляется миграцией.
