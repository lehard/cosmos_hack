// Пакет ops — операции HTTP API модуля ops (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/ops (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-109, FR-127, AD-25, AD-45.
// Владелец после волны 1: эпик 34 (эксплуатация).
package ops
