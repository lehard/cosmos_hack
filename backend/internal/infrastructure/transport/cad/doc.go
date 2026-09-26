// Пакет cad — операции HTTP API модуля cad (зона transport, AD-1, AD-20):
// регистрирует операции Huma с x-ant-action через httpapi.Register и вызывает
// только ведущие порты application/cad (Queries, Commands) — одинаково для
// режимов live и fixtures (AD-36).
//
// Требования: FR-94, AD-18.
// Владелец после волны 1: эпик 31 (Галактика, MES, КОМПАС-3D).
package cad
