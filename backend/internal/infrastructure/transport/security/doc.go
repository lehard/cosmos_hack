// Пакет security — операции HTTP API модуля security (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/security (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-72…FR-77, FR-118, FR-146, AD-8, AD-9, AD-24, AD-28, AD-46.
// Владелец после волны 1: эпик 29 (доверие).
package security
