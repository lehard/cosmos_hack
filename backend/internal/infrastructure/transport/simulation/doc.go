// Пакет simulation — операции HTTP API модуля simulation (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/simulation (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-104…FR-108, FR-119, FR-124, FR-129, FR-152, AD-26, AD-37, AD-38.
// Владелец после волны 1: эпик 32 (симуляция), 36 (цифровой стенд).
package simulation
