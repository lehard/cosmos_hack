// Пакет fixtures — курсор мира заготовок в Postgres (зона storage, AD-36, AD-35:
// ключ fixture_cursor = postgres): положение курсора сценария общее для всех
// копий api — схема fixtures, одна строка на курсор.
//
// Слой: infrastructure/storage. Реализует ведомый порт platform.FixtureCursor;
// зону fixtures не импортирует (AD-1) — курсор подаёт миру заготовок cmd/*
// (loader.SetCursor). Запросы — pgx без sqlc (решение дирижёра Д-21).
// Миграция — migrations/ (goose); EnsureSchema — та же DDL для профилей, где
// роль migrate ещё не применяет миграции модулей.
//
// Владелец: эпик 09.
package fixtures
