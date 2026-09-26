// Пакет materials — операции HTTP API модуля materials (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/materials (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-75, FR-102, FR-139, AD-23.
// Владелец после волны 1: эпик 04 (журнал и хранение).
package materials
