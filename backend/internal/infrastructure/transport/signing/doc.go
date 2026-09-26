// Пакет signing — операции HTTP API модуля signing (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/signing (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-66…FR-70, FR-76, FR-79, AD-10, AD-11, AD-32, AD-33.
// Владелец после волны 1: эпик 27 (подписи и ключи), 05 (криптоядро и генезис).
package signing
