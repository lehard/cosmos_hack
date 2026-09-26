// Пакет world — генератор мира заготовок из компактного описания (служебная
// зона fixtures, AD-36; FR-129, FR-150).
//
// Слой: infrastructure/fixtures. Читает scenarios/fixtures/‹сценарий›/world.yaml
// и нормативный слой (normative/policy, normative/desks) и строит:
//   - модель мира: изделия с маршрутом по step_key, оси статусов (contracts/
//     statuses.yaml), несоответствия, области риска, сообщения 1С, источники,
//     карантин, целостность — снимок на каждый шаг курсора (model.go);
//   - ответы операций contracts/openapi.yaml по шагам — телами типов
//     application/‹модуль› (render.go): тот же тип, что отдаёт live, поэтому
//     ID и статусы совпадают на всех экранах;
//   - файлы scenarios/fixtures/‹сценарий›/{scenario.yaml,steps/NN.yaml},
//     scenarios/fixtures/common/responses.yaml и их встроенную копию
//     infrastructure/fixtures/loader/bundle (generate.go).
//
// Генерация — тестом: `go test ./internal/infrastructure/fixtures/world -update`
// пишет файлы; без -update тест требует, чтобы файлы совпадали с генератором,
// а тела ответов проходили схемы openapi.yaml (make check).
//
// Владелец: эпик 09; после волны 2 — эпик 32 (генератор сценариев).
package world
