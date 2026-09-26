// Пакет reference — операции HTTP API модуля reference (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/reference (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-17, FR-80, FR-81, FR-95, AD-31, AD-18.
// Владелец после волны 1: эпик 19 (справочники, календарь, смены).
package reference
