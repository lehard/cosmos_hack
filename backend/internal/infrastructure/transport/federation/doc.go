// Пакет federation — операции HTTP API модуля federation (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/federation (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-131…FR-134, AD-19.
// Владелец после волны 1: эпик 41 (федерация).
package federation
