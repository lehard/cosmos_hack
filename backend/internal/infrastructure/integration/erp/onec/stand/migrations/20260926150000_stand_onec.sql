-- Схема stand_onec — собственное состояние stand-а 1С (AD-18, NFR-TEST-2):
-- регистр входящих сообщений расширения «КонтрольКачества» (ключ
-- идемпотентности → квитанция), документы 1С, созданные по сообщениям, и
-- этапы производства, созданные на странице stand-а. Это данные
-- «эмулируемой 1С», а не проекции журнала ant: журнал об их содержимом
-- ничего не знает, кроме квитанций (erp.posting.responded).
-- Выполняется ролью ant_owner; роль stands работает ролью ant_app.

-- +goose Up
CREATE SCHEMA stand_onec;
REVOKE ALL ON SCHEMA stand_onec FROM PUBLIC;

CREATE TABLE stand_onec.objects (
    kind    text        NOT NULL CHECK (kind IN ('message', 'document', 'stage')),
    id      text        NOT NULL,
    body    jsonb       NOT NULL,
    created timestamptz NOT NULL,
    updated timestamptz NOT NULL,
    PRIMARY KEY (kind, id)
);

GRANT USAGE ON SCHEMA stand_onec TO ant_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA stand_onec TO ant_app;

-- +goose Down
DROP SCHEMA stand_onec CASCADE;
