// Пакет item — операции HTTP API модуля item (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/item (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-42…FR-46, FR-130, AD-16, AD-41.
// Владелец после волны 1: эпик 18 (изделие, генеалогия).
package item
