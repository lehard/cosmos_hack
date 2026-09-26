# Документы «Главного»

«Главный» — платформа управления качеством производства: доверенная система контроля качества изделий; в коде — `ant`.

| Файл | Что |
|---|---|
| [reviewer-guide.md](reviewer-guide.md) | **Проверяющим.** Путеводитель по критериям оценки кейса: команда, что увидеть, где в коде |
| [presentation.md](presentation.md) | **Материалы защиты.** Техническая мини-презентация (PDF сдаётся на платформе); скринкасты — [раздел README](../README.md#материалы-защиты), ссылки будут добавлены |
| [jury-answers.md](jury-answers.md) | Короткие ответы на частые вопросы жюри со ссылками на документы |
| [sources.md](sources.md) | Единый список источников: кейс, ГОСТ, открытые наборы, документация внешних систем |
| [guides/](guides/README.md) | **Пользователям.** Должностные инструкции ролей на платформе (встроены в интерфейс — «Справка для вашей роли»), администратора, описание демо-сценариев и [расширения подписи «Главный — подпись»](guides/sign_extension.md) (тот же текст — во встроенной справке интерфейса) |
| [architecture.md](architecture.md) | **Разработчикам — начать здесь.** Архитектура и карта всех архитектурных документов |
| [prd.md](prd.md) | Требования к системе (PRD), копия для жюри; FR-1…FR-158, решения этапа разработки и разбора — §11.20–11.23 |
| [architecture-spine.md](architecture-spine.md) | Архитектурный спайн: парадигма, решения AD-1…AD-47, стек, дерево репозитория |
| [case-compliance.md](case-compliance.md) | **Проверяющим:** каждый пункт кейса и критерий оценки → архитектурное решение → код и проверка → статус как есть |
| [data-model.md](data-model.md) | Модель данных: запись журнала, сущности, статусы, проекции |
| [specifications.md](specifications.md), [codegen.md](codegen.md) | Спецификации API, событий, BPMN; кодогенерация и проверки контрактов |
| [scenario-processing.md](scenario-processing.md) | Как система отрабатывает сценарии кейса |
| [integrations/README.md](integrations/README.md) | 1С, Галактика:ERP, MES, КОМПАС-3D, СКУД; stand-ы и экран «Интеграции» |
| [sso.md](sso.md) | Единый вход (SSO) — перспектива: OIDC / SAML / LDAP адаптером порта `IdentityProvider` |
| [threat-model.md](threat-model.md), [crypto.md](crypto.md) | Модель угроз, меры приказа ФСТЭК № 117; подписи, ключи, криптопрофили |
| [scaling.md](scaling.md), [observability-kafka-otel.md](observability-kafka-otel.md), [backup-restore.md](backup-restore.md) | Масштабирование; OpenTelemetry и Kafka; резервирование и восстановление |
| [vision-camera-project.md](vision-camera-project.md), [target-components.md](target-components.md), [federation.md](federation.md) | Комплекс «камеры + ИИ»; компоненты кейса §3.2; межзаводская кооперация |
| [document-catalog.md](document-catalog.md), [normative-anchors.md](normative-anchors.md), [bpmn-ext-properties.md](bpmn-ext-properties.md) | Документы по этапам; нормативные опоры (ГОСТ); свойства расширения BPMN |
| [assumptions.md](assumptions.md), [third-party.md](third-party.md), [glossary.md](glossary.md) | Ограничения и допущения; заимствованный код и лицензии; термины |
| [process/](process/README.md) | Материалы процессной сессии: наши находки, паспорт КТ-3, методы обнаружения, следующее наблюдение, предложения, фабрика документов, паспорт изделия, экран линии, краевой агент, схема v0.3.2; процессный набор сценариев — [../scenarios/process-session/README.md](../scenarios/process-session/README.md), эталонные сообщения внешних систем — [integrations/examples/README.md](integrations/examples/README.md) |
| `licenses/` | Перечень зависимостей с лицензиями — `make licenses` (NFR-SEC-2) |
| `scripts/check-docs.mjs` | Проверка документации: ссылки и якоря, руководства = справка, пути в путеводителе, заголовки пакетов Go (NFR-DOC-3) |

Копии PRD и спайна обновляются из рабочей папки планирования и называют систему рабочим именем `ant`; источник
правды в коде — сам код.
