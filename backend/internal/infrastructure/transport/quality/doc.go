// Пакет quality — операции HTTP API модуля quality (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/quality (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-14, FR-35…FR-38, FR-48, FR-125, FR-144, AD-3, AD-27, AD-29, AD-30.
// Владелец после волны 1: эпик 20 (качество).
package quality
