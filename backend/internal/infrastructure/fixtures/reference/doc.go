// Пакет reference — заготовки ведущих портов модуля reference (служебная зона fixtures,
// AD-36, FR-150): отвечает на те же Queries и Commands, что и live, данными
// сценариев scenarios/fixtures/‹сценарий›/steps/NN.yaml поверх общего загрузчика
// (infrastructure/fixtures/loader) и курсора FixtureCursor. Таблиц проекций не
// трогает, в журнал не пишет, наружу не отправляет; права, гарды и допустимые
// действия — общий декоратор (application/access.Gate), здесь их нет.
//
// Владелец: эпик 09 (мир заготовок), после волны 2 — эпик 19 (справочники, календарь, смены).
package reference
