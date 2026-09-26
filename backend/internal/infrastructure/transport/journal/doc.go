// Пакет journal — операции HTTP API модуля journal (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/journal (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-40, FR-71, FR-72, FR-122, AD-2, AD-8, AD-37, AD-44, AD-45.
// Владелец после волны 1: эпик 04 (журнал и хранение), 29 (хранитель).
package journal
