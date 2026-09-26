// Пакет documents — операции HTTP API модуля documents (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/documents (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-65, FR-136, FR-139, AD-12, AD-13, AD-43.
// Владелец после волны 1: эпик 28 (документы), 44 (каталог).
package documents
