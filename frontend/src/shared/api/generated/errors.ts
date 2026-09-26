// СГЕНЕРИРОВАНО frontend/scripts/generate.mjs — руками не править (AD-20).
// Источник: contracts/errors.yaml

/** Каталог кодов ошибок: код → HTTP-статус, заголовок и ключ текста интерфейса. */
export const errorCatalog = {
  "api.not_implemented": {
    "status": 501,
    "title": "Операция ещё не реализована",
    "uiKey": "errors.generic"
  },
  "api.validation_failed": {
    "status": 400,
    "title": "Запрос не соответствует контракту",
    "uiKey": "errors.generic"
  },
  "api.not_found": {
    "status": 404,
    "title": "Объект не найден",
    "uiKey": "empty.notFound"
  },
  "api.replay_read_only": {
    "status": 403,
    "title": "В режиме воспроизведения действия недоступны",
    "uiKey": "errors.decision.replayReadOnly"
  },
  "api.internal_error": {
    "status": 500,
    "title": "Внутренняя ошибка",
    "uiKey": "errors.generic"
  },
  "api.rate_limited": {
    "status": 429,
    "title": "Слишком много запросов",
    "uiKey": "errors.generic"
  },
  "ingest.unknown_schema_version": {
    "status": 422,
    "title": "Неизвестная версия контракта",
    "uiKey": "errors.ingest.unknownSchemaVersion"
  },
  "ingest.missing_required_field": {
    "status": 422,
    "title": "Нет обязательного поля",
    "uiKey": "errors.ingest.missingRequiredField"
  },
  "ingest.unknown_enum_value": {
    "status": 202,
    "title": "Неизвестное значение перечисления — принято с пометкой",
    "uiKey": "errors.ingest.unknownEnumValueAccepted"
  },
  "ingest.unknown_enum_value_critical": {
    "status": 422,
    "title": "Неизвестное значение в поле, важном для безопасности",
    "uiKey": "errors.ingest.unknownEnumValueQuarantined"
  },
  "ingest.schema_violation": {
    "status": 422,
    "title": "Сообщение не соответствует схеме",
    "uiKey": "errors.ingest.incompatibleChange"
  },
  "ingest.unknown_event_type": {
    "status": 422,
    "title": "Неизвестный тип события",
    "uiKey": "errors.ingest.incompatibleChange"
  },
  "ingest.canonical_form_violation": {
    "status": 422,
    "title": "Подписанные байты не в каноническом виде",
    "uiKey": "errors.ingest.signatureInvalid"
  },
  "ingest.signature_invalid": {
    "status": 422,
    "title": "Подпись сообщения недействительна",
    "uiKey": "errors.ingest.signatureInvalid"
  },
  "ingest.unknown_source": {
    "status": 403,
    "title": "Источник не зарегистрирован",
    "uiKey": "errors.ingest.unknownSource"
  },
  "ingest.revoked_key": {
    "status": 403,
    "title": "Ключ источника отозван",
    "uiKey": "errors.ingest.revokedKey"
  },
  "ingest.duplicate_conflict": {
    "status": 409,
    "title": "Тот же номер события, другое содержимое",
    "uiKey": "errors.ingest.duplicateConflict"
  },
  "ingest.batch_too_large": {
    "status": 413,
    "title": "Слишком большая пачка",
    "uiKey": "errors.generic"
  },
  "journal.stale_state": {
    "status": 409,
    "title": "Состояние изменилось после проверки",
    "uiKey": "errors.staleState"
  },
  "journal.stale_policy": {
    "status": 409,
    "title": "Политика доступа изменилась",
    "uiKey": "errors.staleState"
  },
  "journal.concession_exhausted": {
    "status": 409,
    "title": "Лимит разрешения на отклонение исчерпан",
    "uiKey": "errors.decision.concessionRequired"
  },
  "journal.duplicate": {
    "status": 409,
    "title": "Запись уже есть в журнале",
    "uiKey": "errors.generic"
  },
  "journal.append_only": {
    "status": 405,
    "title": "Журнал только на дописывание",
    "uiKey": "errors.generic"
  },
  "journal.fenced": {
    "status": 503,
    "title": "Запись отвергнута: аренда партиции утрачена",
    "uiKey": "errors.generic"
  },
  "journal.time_regression": {
    "status": 500,
    "title": "Убывание времени записи",
    "uiKey": "errors.generic"
  },
  "access.unauthenticated": {
    "status": 401,
    "title": "Нужен вход",
    "uiKey": "errors.access.sessionExpired"
  },
  "access.login_failed": {
    "status": 401,
    "title": "Неверный логин или пароль",
    "uiKey": "errors.access.loginFailed"
  },
  "access.account_pending": {
    "status": 403,
    "title": "Заявка не активирована",
    "uiKey": "errors.access.accountPending"
  },
  "access.forbidden": {
    "status": 403,
    "title": "Нет полномочий на это действие",
    "uiKey": "errors.access.forbidden"
  },
  "access.signature_required": {
    "status": 403,
    "title": "Нужна подпись уполномоченного",
    "uiKey": "errors.access.needSignature"
  },
  "access.wrong_workplace": {
    "status": 403,
    "title": "Не своё рабочее место",
    "uiKey": "errors.access.wrongWorkplace"
  },
  "access.self_grant": {
    "status": 403,
    "title": "Выдача прав себе",
    "uiKey": "errors.access.selfGrant"
  },
  "access.not_qualified": {
    "status": 409,
    "title": "Нет действующей квалификации",
    "uiKey": "errors.decision.qualificationExpired"
  },
  "access.controller_approval_required": {
    "status": 403,
    "title": "Нужно согласование начальника ОТК",
    "uiKey": "errors.access.needSignature"
  },
  "access.not_in_zone": {
    "status": 403,
    "title": "Нет в зоне по СКУД",
    "uiKey": "errors.access.notInZone"
  },
  "access.admission_denied": {
    "status": 403,
    "title": "Отказ в допуске к рабочему месту",
    "uiKey": "errors.access.forbidden"
  },
  "access.no_stamp": {
    "status": 403,
    "title": "Нет действующего цифрового клейма",
    "uiKey": "errors.decision.noStamp"
  },
  "access.separation_of_duties": {
    "status": 403,
    "title": "Разделение обязанностей",
    "uiKey": "errors.decision.separationOfDuties"
  },
  "signing.agent_not_found": {
    "status": 424,
    "title": "Агент токена не найден",
    "uiKey": "errors.signing.agentNotFound"
  },
  "signing.token_missing": {
    "status": 424,
    "title": "Токен не вставлен",
    "uiKey": "errors.signing.tokenMissing"
  },
  "signing.pin_wrong": {
    "status": 424,
    "title": "Неверный ПИН-код",
    "uiKey": "errors.signing.pinWrong"
  },
  "signing.key_revoked": {
    "status": 422,
    "title": "Ключ отозван",
    "uiKey": "errors.signing.certRevoked"
  },
  "signing.key_unavailable": {
    "status": 422,
    "title": "Ключ недоступен — проверить подпись нельзя",
    "uiKey": "errors.signing.keyUnavailable"
  },
  "signing.foreign_key": {
    "status": 403,
    "title": "Чужой ключ",
    "uiKey": "errors.access.forbidden"
  },
  "signing.profile_downgrade": {
    "status": 422,
    "title": "Понижение криптопрофиля",
    "uiKey": "errors.signing.packageTampered"
  },
  "signing.package_tampered": {
    "status": 422,
    "title": "Пакет изменён",
    "uiKey": "errors.signing.packageTampered"
  },
  "signing.document_changed": {
    "status": 409,
    "title": "Документ изменён после подписи",
    "uiKey": "errors.signing.documentChanged"
  },
  "signing.qr_mismatch": {
    "status": 422,
    "title": "QR-отпечаток на скане не совпадает с документом",
    "uiKey": "errors.signing.qrMismatch"
  },
  "signing.attester_is_signer": {
    "status": 422,
    "title": "Заверитель совпадает с подписантом",
    "uiKey": "errors.signing.attesterIsSigner"
  },
  "signing.paper_forbidden": {
    "status": 422,
    "title": "Бумага на этом этапе запрещена",
    "uiKey": "errors.signing.paperForbidden"
  },
  "signing.no_signature_path": {
    "status": 422,
    "title": "Нет пути подписи",
    "uiKey": "errors.signing.noSignaturePath"
  },
  "signing.level_not_allowed": {
    "status": 422,
    "title": "Уровень подписи не допускается",
    "uiKey": "errors.generic"
  },
  "signing.unknown_doc_format": {
    "status": 422,
    "title": "Неизвестная версия формата документа",
    "uiKey": "errors.generic"
  },
  "document.stage_not_open": {
    "status": 409,
    "title": "Этап маршрута не ждёт подписи",
    "uiKey": "errors.generic"
  },
  "nonconformity.concession_required": {
    "status": 422,
    "title": "Нужно действующее разрешение на отклонение",
    "uiKey": "errors.decision.concessionRequired"
  },
  "nonconformity.reject_reason_required": {
    "status": 422,
    "title": "Нужна причина отклонения",
    "uiKey": "errors.decision.rejectReasonRequired"
  },
  "nonconformity.return_only_unprocessed": {
    "status": 422,
    "title": "Вернуть поставщику можно только необработанное",
    "uiKey": "errors.decision.returnOnlyUnprocessed"
  },
  "nonconformity.approvals_missing": {
    "status": 409,
    "title": "Не хватает подписей",
    "uiKey": "errors.decision.approvalsMissing"
  },
  "nonconformity.item_blocked": {
    "status": 409,
    "title": "Изделие заблокировано",
    "uiKey": "errors.decision.itemBlocked"
  },
  "nonconformity.gate_without_signature": {
    "status": 403,
    "title": "Точка предъявления без подписи",
    "uiKey": "errors.decision.gateWithoutSignature"
  },
  "nonconformity.method_result_missing": {
    "status": 409,
    "title": "Нет результата метода контроля",
    "uiKey": "empty.inspectionMissing"
  },
  "nonconformity.intervention_open": {
    "status": 409,
    "title": "Открыто вмешательство",
    "uiKey": "errors.decision.interventionOpen"
  },
  "nonconformity.concession_not_applicable": {
    "status": 422,
    "title": "Разрешение на отклонение не применимо",
    "uiKey": "errors.decision.concessionRequired"
  },
  "nonconformity.invalid_transition": {
    "status": 409,
    "title": "Решение недопустимо в этом состоянии",
    "uiKey": "errors.generic"
  },
  "nonconformity.process_hold_not_active": {
    "status": 409,
    "title": "Остановка точки процесса не действует",
    "uiKey": "errors.generic"
  },
  "item.not_registered": {
    "status": 409,
    "title": "Изделие не зарегистрировано",
    "uiKey": "errors.generic"
  },
  "item.already_registered": {
    "status": 409,
    "title": "Изделие уже зарегистрировано",
    "uiKey": "errors.generic"
  },
  "item.carrier_in_use": {
    "status": 409,
    "title": "Носитель уже действует у другого изделия",
    "uiKey": "errors.generic"
  },
  "item.carrier_not_active": {
    "status": 409,
    "title": "Носитель не нанесён",
    "uiKey": "errors.generic"
  },
  "item.identification_not_questioned": {
    "status": 409,
    "title": "Идентификация не под сомнением",
    "uiKey": "errors.generic"
  },
  "item.identification_questioned": {
    "status": 409,
    "title": "Идентификация изделия под сомнением",
    "uiKey": "errors.generic"
  },
  "item.intervention_not_open": {
    "status": 409,
    "title": "Вмешательство не открыто",
    "uiKey": "errors.generic"
  },
  "item.zone_unknown": {
    "status": 422,
    "title": "Зона не описана в КД",
    "uiKey": "errors.generic"
  },
  "item.assembly_cycle": {
    "status": 409,
    "title": "Сборка образует цикл",
    "uiKey": "errors.generic"
  },
  "item.component_already_assembled": {
    "status": 409,
    "title": "Компонент уже в другой сборке",
    "uiKey": "errors.generic"
  },
  "item.already_released": {
    "status": 409,
    "title": "Изделие уже выпущено",
    "uiKey": "errors.generic"
  },
  "item.lot_not_accepted": {
    "status": 409,
    "title": "Партия не принята",
    "uiKey": "errors.generic"
  },
  "item.lot_already_registered": {
    "status": 409,
    "title": "Партия уже зарегистрирована",
    "uiKey": "errors.generic"
  },
  "item.group_invalid": {
    "status": 422,
    "title": "Группа изделий составлена неверно",
    "uiKey": "errors.generic"
  },
  "item.group_not_active": {
    "status": 409,
    "title": "Группа не действует",
    "uiKey": "errors.generic"
  },
  "item.binding_subject_unknown": {
    "status": 404,
    "title": "Привязываемое событие не найдено",
    "uiKey": "errors.generic"
  },
  "process.unsupported_element": {
    "status": 422,
    "title": "Неподдерживаемый элемент BPMN",
    "uiKey": "errors.process.unsupportedElement"
  },
  "process.schema_violation": {
    "status": 422,
    "title": "Нарушение схемы расширения",
    "uiKey": "errors.process.schemaViolation"
  },
  "process.step_key_missing": {
    "status": 422,
    "title": "У элемента нет step_key",
    "uiKey": "errors.process.schemaViolation"
  },
  "process.unreachable_node": {
    "status": 422,
    "title": "Недостижимый узел",
    "uiKey": "errors.process.unreachableNode"
  },
  "process.closing_path_without_human": {
    "status": 422,
    "title": "Закрывающий путь без контроля человеком",
    "uiKey": "errors.process.closingPathWithoutHuman"
  },
  "process.presentation_point_without_role": {
    "status": 422,
    "title": "Точка предъявления без роли или полномочия",
    "uiKey": "errors.process.presentationPointWithoutRole"
  },
  "process.inspection_without_requirement": {
    "status": 200,
    "title": "Контроль без ссылки на требование КД",
    "uiKey": "errors.process.inspectionWithoutRequirement"
  },
  "process.version_tampered": {
    "status": 409,
    "title": "Содержимое действующей версии изменено",
    "uiKey": "errors.process.versionTampered"
  },
  "process.version_unknown": {
    "status": 409,
    "title": "Версия процесса изделия не найдена",
    "uiKey": "errors.process.versionTampered"
  },
  "process.quorum_incomplete": {
    "status": 409,
    "title": "Не хватает подписей кворума",
    "uiKey": "errors.process.quorumIncomplete"
  },
  "process.precondition_failed": {
    "status": 409,
    "title": "Не выполнено предусловие операции",
    "uiKey": "errors.decision.preconditionFailed"
  },
  "process.qualification_expired": {
    "status": 409,
    "title": "Квалификация исполнителя истекла",
    "uiKey": "errors.decision.qualificationExpired"
  },
  "process.zone_check_required": {
    "status": 409,
    "title": "Сначала проверка зоны",
    "uiKey": "errors.decision.zoneCheckRequired"
  },
  "process.rework_limit_exceeded": {
    "status": 422,
    "title": "Лимит доработок зоны исчерпан",
    "uiKey": "errors.decision.reworkLimitReached"
  },
  "process.implicit_merge": {
    "status": 422,
    "title": "Неявное слияние стрелок без шлюза",
    "uiKey": "errors.process.schemaViolation"
  },
  "process.condition_invalid": {
    "status": 422,
    "title": "Условие на стрелке не по языку условий",
    "uiKey": "errors.process.schemaViolation"
  },
  "process.status_expired": {
    "status": 409,
    "title": "Истёк срок статуса",
    "uiKey": "errors.decision.statusExpired"
  },
  "incident.auto_exclude_forbidden": {
    "status": 403,
    "title": "Автоматически исключить нельзя",
    "uiKey": "errors.decision.autoExcludeForbidden"
  },
  "incident.basis_required": {
    "status": 422,
    "title": "Нужно основание",
    "uiKey": "riskScope.basisRequired"
  },
  "incident.item_not_in_scope": {
    "status": 409,
    "title": "Изделия нет в области риска",
    "uiKey": "errors.generic"
  },
  "incident.closed": {
    "status": 409,
    "title": "Инцидент закрыт",
    "uiKey": "errors.generic"
  },
  "incident.effectiveness_plan_required": {
    "status": 422,
    "title": "Нужен план проверки эффективности",
    "uiKey": "errors.generic"
  },
  "incident.action_state": {
    "status": 409,
    "title": "Мера не в том состоянии",
    "uiKey": "errors.generic"
  },
  "incident.suggestion_state": {
    "status": 409,
    "title": "Предложение уже решено",
    "uiKey": "errors.generic"
  },
  "incident.explanation_required": {
    "status": 422,
    "title": "Нужно письменное объяснение работника",
    "uiKey": "errors.generic"
  },
  "erp.unavailable": {
    "status": 503,
    "title": "1С недоступна",
    "uiKey": "errors.integration.erpUnavailable"
  },
  "erp.data_error": {
    "status": 422,
    "title": "1С вернула ошибку данных",
    "uiKey": "errors.integration.erpDataError"
  },
  "erp.id_mapping_missing": {
    "status": 422,
    "title": "Нет соответствия идентификатора",
    "uiKey": "errors.integration.idMappingMissing"
  },
  "erp.contract_not_found": {
    "status": 422,
    "title": "Не найден договор с контрагентом",
    "uiKey": "errors.integration.contractNotFound"
  },
  "erp.no_ack": {
    "status": 504,
    "title": "1С не подтвердила приём",
    "uiKey": "errors.integration.noAck"
  },
  "erp.duplicate_in_erp": {
    "status": 200,
    "title": "1С уже провела документ",
    "uiKey": "errors.integration.duplicateInErp"
  },
  "erp.contract_incompatible": {
    "status": 422,
    "title": "Несовместимое изменение контракта обмена",
    "uiKey": "errors.integration.contractIncompatible"
  },
  "erp.not_quarantined": {
    "status": 409,
    "title": "Сообщение не ждёт решения",
    "uiKey": "errors.integration.erpDataError"
  },
  "erp.correction_pending": {
    "status": 409,
    "title": "Новая версия отправленного сообщения ждёт решения",
    "uiKey": "errors.integration.erpDataError"
  },
  "erp.channel_degraded": {
    "status": 503,
    "title": "Канал обмена в режиме degraded",
    "uiKey": "errors.integration.contractIncompatible"
  },
  "erp.galaktika_unavailable": {
    "status": 503,
    "title": "Галактика:ERP недоступна",
    "uiKey": "errors.integration.galaktikaUnavailable"
  },
  "mes.unavailable": {
    "status": 503,
    "title": "MES недоступна",
    "uiKey": "errors.integration.mesUnavailable"
  },
  "analyzer.no_qualified_analyzer": {
    "status": 200,
    "title": "Нет допущенного анализатора",
    "uiKey": "errors.vision.noQualifiedAnalyzer"
  },
  "analyzer.reinstate_requires_head_of_qc": {
    "status": 403,
    "title": "Вернуть анализатор может только начальник ОТК",
    "uiKey": "errors.vision.returnRequiresHeadOfQc"
  },
  "analyzer.trust_level_exceeded": {
    "status": 403,
    "title": "Действие сверх уровня доверия паспорта",
    "uiKey": "errors.generic"
  },
  "analyzer.invalid_transition": {
    "status": 409,
    "title": "Действие с паспортом недоступно в его статусе",
    "uiKey": "errors.generic"
  },
  "analyzer.admission_route_open": {
    "status": 409,
    "title": "Протокол допуска не подписан",
    "uiKey": "errors.generic"
  },
  "federation.extract_tampered": {
    "status": 422,
    "title": "Выписка паспорта изменена",
    "uiKey": "errors.federation.extractTampered"
  },
  "federation.extract_unverifiable": {
    "status": 202,
    "title": "Происхождение не подтверждено",
    "uiKey": "errors.federation.extractUnverifiable"
  },
  "reference.not_found": {
    "status": 422,
    "title": "Запись справочника не найдена",
    "uiKey": "errors.integration.refNotFound"
  },
  "simulation.scenario_keys_in_prod": {
    "status": 500,
    "title": "Ключи сценариев в профиле prod",
    "uiKey": "errors.generic"
  },
  "simulation.run_not_found": {
    "status": 404,
    "title": "Прогон не найден",
    "uiKey": "empty.scenarioNotStarted"
  }
} as const

/** Код ошибки из каталога. */
export type ErrorCode = keyof typeof errorCatalog
