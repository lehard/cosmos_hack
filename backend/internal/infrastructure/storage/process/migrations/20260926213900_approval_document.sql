-- Лист утверждения версии процесса (эпик 39, FR-23, AD-43): документ с
-- маршрутом кворума, заведённый при отправке версии на утверждение.

-- +goose Up
ALTER TABLE process.versions ADD COLUMN approval_document_id text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE process.versions DROP COLUMN approval_document_id;
