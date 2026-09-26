// Пакет crossitem — операции HTTP API модуля crossitem (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/crossitem (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-15, FR-45, FR-121, FR-151, AD-41, AD-42.
// Владелец после волны 1: эпик 07 (рамка), 18 (генеалогия).
package crossitem
