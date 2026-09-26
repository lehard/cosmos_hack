// Пакет cad — производственные сценарии модуля cad слоя application: импорт
// условной сборки КОМПАС-3D (FR-94, PRD §11.15, AD-18).
//
// Слой: application (AD-1). Объявляет ведущие порты Queries и Commands
// (реализации: live — Service здесь, fixtures — infrastructure/fixtures/cad,
// AD-36) и ведомые порты модуля: Reader (формат файла → сборка на нашем
// языке; адаптер infrastructure/integration/cad/kompas), Intake (обычный
// приём фактов), Nomenclature (номенклатура учётной системы для сверки
// обозначений). Вызывает domain/cad. HTTP-фреймворка и драйверов БД здесь
// нет: операции регистрирует infrastructure/transport/cad.
//
// Путь импорта: файл → Reader (схема contracts/integrations/cad/assembly.schema.json,
// версия формата) → domain/cad.Translate (дерево, зоны, ограничения,
// соответствия, расхождения) → факты через обычный приём: cad.assembly.imported
// и reference.external_id.mapped (источник cad.kompas, вид — импорт) →
// проекция cad.assembly (роль projector) → операция cad.assembly.list.
// Тот же файл повторно — дубль (event_id от отпечатка файла, AD-7).
//
// Требования: FR-94, FR-95, AD-17, AD-18, AD-31.
// Владелец: эпик 31 (Галактика, MES, КОМПАС-3D).
package cad
