// Пакет analytics — операции HTTP API модуля analytics (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/analytics (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-3, FR-5, FR-86…FR-89, AD-45.
// Владелец после волны 1: эпик 25 (аналитика).
package analytics
