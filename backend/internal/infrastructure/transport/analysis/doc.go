// Пакет analysis — операции HTTP API модуля analysis (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/analysis (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-58…FR-64, FR-135, FR-138, FR-143, FR-153, AD-3, AD-29, AD-42.
// Владелец после волны 1: эпик 22 (разбор и область риска), 42 (предложения).
package analysis
