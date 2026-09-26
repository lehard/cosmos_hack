// Пакет process — операции HTTP API модуля process (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/process (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-10…FR-25, FR-44, FR-47, FR-125, FR-154…FR-156, AD-17, AD-13, AD-43.
// Владелец после волны 1: эпик 17 (процесс), 39 (редактор и кворум).
package process
