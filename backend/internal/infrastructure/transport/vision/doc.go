// Пакет vision — операции HTTP API модуля vision (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/vision (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-97…FR-103, FR-126, AD-18, AD-29.
// Владелец после волны 1: эпик 33 (порты и эмуляторы), 40 (адаптация).
package vision
