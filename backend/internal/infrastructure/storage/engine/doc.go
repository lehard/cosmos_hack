// Пакет engine — хранение движка (зона storage, AD-1): своя схема Postgres
// «engine» (миграции goose в migrations/, версии — метки времени) с
// проекциями каркаса, вкладами изделий в показатели и журналом изменений для
// живых обновлений; адаптеры портов application/engine ProjectionStore и
// ChangeLog (LISTEN/NOTIFY, AD-6) и применение эффектов движка внутри
// транзакции journal.Append (AD-44, AD-45).
//
// Решение эпика 07: запросы простые и статические — на pgx без sqlc; при
// росте — sqlc.yaml рядом (make generate подхватит).
//
// Слой: infrastructure/storage; реализует ведомые порты application/engine;
// не импортирует другие зоны.
// Владелец: эпик 07 (движок, воркер, проекции).
package engine
