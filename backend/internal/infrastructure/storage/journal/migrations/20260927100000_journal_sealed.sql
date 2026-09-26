-- Шифрование при хранении (AD-23, FR-75; эпик 29): блок записи {salt,
-- конверт DSSE} шифруется AEAD ключом DEK; обёртки DEK ключом KEK — в
-- journal.dek_wraps (только дописывание, вне цепочки). KEK — в отдельном томе,
-- в БД его нет: копия базы без тома KEK содержимого записей не выдаёт.
--
-- Столбцы записи: dek_id и aead — NULL у открытых записей (демо-трек до
-- эпика 29 и хранилище без KEK); у зашифрованных salt пуст (соль внутри
-- блока), envelope — шифротекст с тегом, nonce — нонс AEAD. Звено и commit
-- не меняются: commit = H(salt ‖ JCS(конверт)) считается до шифрования.

-- +goose Up
ALTER TABLE journal.entries ADD COLUMN dek_id text;
ALTER TABLE journal.entries ADD COLUMN aead text CHECK (aead IN ('aes_256_gcm', 'kuznyechik_mgm'));
ALTER TABLE journal.entries ADD COLUMN nonce bytea;

-- Обёртки DEK (конвертная схема): ротация KEK — новые обёртки тем же DEK и
-- служебная запись; старые обёртки остаются (дописывание).
CREATE TABLE journal.dek_wraps (
    dek_id     text        NOT NULL,
    kek_id     text        NOT NULL,
    wrapped    bytea       NOT NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (dek_id, kek_id)
);

CREATE TRIGGER dek_wraps_forbid_change BEFORE UPDATE OR DELETE ON journal.dek_wraps
    FOR EACH ROW EXECUTE FUNCTION journal.forbid_change();
CREATE TRIGGER dek_wraps_forbid_truncate BEFORE TRUNCATE ON journal.dek_wraps
    FOR EACH STATEMENT EXECUTE FUNCTION journal.forbid_change();
ALTER TABLE journal.dek_wraps ENABLE ALWAYS TRIGGER dek_wraps_forbid_change;
ALTER TABLE journal.dek_wraps ENABLE ALWAYS TRIGGER dek_wraps_forbid_truncate;

GRANT SELECT, INSERT ON journal.dek_wraps TO ant_app;
GRANT SELECT ON journal.dek_wraps TO ant_verifier;

-- +goose Down
DROP TABLE journal.dek_wraps;
ALTER TABLE journal.entries DROP COLUMN nonce;
ALTER TABLE journal.entries DROP COLUMN aead;
ALTER TABLE journal.entries DROP COLUMN dek_id;
