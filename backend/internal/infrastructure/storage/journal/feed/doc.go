// Пакет feed — адаптер потребителей журнала (порт application/journal.Consumer,
// AD-45): курсор consumer_offsets(имя, партиция, seq) обновляется в той же
// транзакции, что и выход потребителя; охват partition | global; сигнал «есть
// новое» — LISTEN/NOTIFY с seq (AD-6).
//
// Слой: infrastructure/storage. Владелец: эпик 04 (журнал), 07 (движок).
package feed
