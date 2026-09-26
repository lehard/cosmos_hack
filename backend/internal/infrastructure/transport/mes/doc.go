// Пакет mes — операции HTTP API модуля mes (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/mes (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-92, FR-93, AD-18.
// Владелец после волны 1: эпик 31 (Галактика, MES, КОМПАС-3D).
package mes
