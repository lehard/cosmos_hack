// Пакет erp — операции HTTP API модуля erp (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/erp (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-90, FR-91, FR-95, FR-96, FR-111, AD-7, AD-18.
// Владелец после волны 1: эпик 30 (1С), 31 (Галактика).
package erp
