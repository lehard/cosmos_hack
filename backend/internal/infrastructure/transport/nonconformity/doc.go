// Пакет nonconformity — операции HTTP API модуля nonconformity (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/nonconformity (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-49…FR-56, FR-144, FR-151, AD-27, AD-30, AD-39, AD-43.
// Владелец после волны 1: эпик 21 (несоответствия, решения, сдерживание).
package nonconformity
