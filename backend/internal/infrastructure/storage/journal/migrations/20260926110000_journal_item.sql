-- Чтение всех записей изделия (ReadQuery.ItemID): вход свёртки изделия и
-- записанные реакции в любых потоках (AD-5). Интеграция эпиков 04 и 07.

-- +goose Up
CREATE INDEX entries_item ON journal.entries (item_id, seq) WHERE chain = 'main' AND item_id IS NOT NULL;

-- +goose Down
DROP INDEX journal.entries_item;
