// Пакет ingest — операции HTTP API модуля ingest (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/ingest (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-26…FR-41, FR-123, FR-140, FR-141, AD-7, AD-18, AD-20, AD-41.
// Владелец после волны 1: эпик 06 (приём и edge-агент).
package ingest
