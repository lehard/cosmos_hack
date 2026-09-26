// Пакет notifications — операции HTTP API модуля notifications (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/notifications (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-8, FR-55, FR-57, AD-4, AD-45.
// Владелец после волны 1: эпик 24 (уведомления).
package notifications
