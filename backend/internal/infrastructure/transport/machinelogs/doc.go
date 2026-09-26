// Пакет machinelogs — операции HTTP API модуля machinelogs (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/machinelogs (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-121, FR-147…FR-149, FR-151, AD-29, AD-42.
// Владелец после волны 1: эпик 23 (MachineLogs).
package machinelogs
