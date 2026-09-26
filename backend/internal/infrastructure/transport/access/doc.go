// Пакет access — операции HTTP API модуля access (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/access (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: .
// Владелец после волны 1: эпик 08, 26, 37.
package access
