/**
 * Решения контролёра и решения по несоответствию (FR-52, FR-53, FR-54, FR-146):
 * какие действия есть, какой операцией API (x-ant-action, AD-40) и каким типом
 * записи журнала каждое исполняется, что обязательно перед подписью, что
 * показать в сводке окна подписи уровня 2 (FR-66, AD-13).
 *
 * Права интерфейс не вычисляет (FR-85): доступно ли действие, говорит сервер
 * (`access.permission.list` по объекту). Здесь — только то, без чего команду
 * заведомо отклонит гард: причина отклонения (`nonconformity.reject_reason_required`),
 * разрешение на отклонение для ремонта и «как есть»
 * (`nonconformity.concession_required`), «вернуть поставщику» только для
 * необработанного (`nonconformity.return_only_unprocessed`).
 */
import { SUMMARY_MAX, type SummaryField } from '@/entities/document'
import { METHOD_TEXT } from './texts'
import type { ConcessionOption, Disposition, InspectionMethod, NcCard } from './types'

/** Действие контролёра по сигналу (FR-52). */
export type SignalAction = 'confirm_nc' | 'reject_signal' | 'request_recheck' | 'accept_and_pass' | 'isolate'

/** Любое действие панели решений. */
export type DecisionAction = SignalAction | 'disposition'

/** Описание действия: операция API, тип записи, текст кнопки «действие — направление». */
export interface DecisionActionDef {
  /** Ожидаемый x-ant-action id операции (AD-40) — ключ прав и объяснения. */
  operation: string
  /** Тип записи журнала, которую пишет операция (каталог AD-40). */
  eventType: string
  /** Ключ текста кнопки. */
  labelKey: string
  /** Причина обязательна до подписи. */
  reasonRequired: boolean
  /** Уровень подписи: все решения контролёра — не ниже 2 (PRD §11.16, AD-13). */
  signatureLevel: 2
}

/** Действия по сигналу (FR-52) — в порядке кнопок. */
export const SIGNAL_ACTIONS: Record<SignalAction, DecisionActionDef> = {
  confirm_nc: {
    operation: 'nonconformity.nonconformity.confirm',
    eventType: 'decision.nonconformity.confirmed',
    labelKey: 'decisions.signal.confirmNc',
    reasonRequired: false,
    signatureLevel: 2,
  },
  // FR-52: отклонение без причины невозможно; исходный сигнал не меняется.
  reject_signal: {
    operation: 'nonconformity.signal.reject',
    eventType: 'decision.signal.rejected',
    labelKey: 'decisions.signal.rejectSignal',
    reasonRequired: true,
    signatureLevel: 2,
  },
  // Изделие не маршрутизируется автоматически — ждёт в изоляции (FR-52).
  request_recheck: {
    operation: 'nonconformity.recheck.request',
    eventType: 'decision.recheck.requested',
    labelKey: 'decisions.signal.requestRecheck',
    reasonRequired: false,
    signatureLevel: 2,
  },
  accept_and_pass: {
    operation: 'nonconformity.presentation.resolve',
    eventType: 'decision.presentation.resolved',
    labelKey: 'decisions.signal.acceptAndPass',
    reasonRequired: false,
    signatureLevel: 2,
  },
  isolate: {
    operation: 'nonconformity.item.isolate',
    eventType: 'decision.item.isolated',
    labelKey: 'decisions.signal.isolate',
    reasonRequired: false,
    signatureLevel: 2,
  },
}

/** Решение по несоответствию (FR-53): переделка / ремонт / как есть / списать / вернуть. */
export const DISPOSITION_ACTION: DecisionActionDef = {
  operation: 'nonconformity.disposition.set',
  eventType: 'decision.disposition.set',
  labelKey: 'decisions.disposition.title',
  // В контракте `reason` у decision.disposition.set обязателен; решение необратимое.
  reasonRequired: true,
  signatureLevel: 2,
}

/** Порядок кнопок действий по сигналу. */
export const SIGNAL_ACTION_ORDER: readonly SignalAction[] = ['confirm_nc', 'reject_signal', 'request_recheck', 'accept_and_pass', 'isolate']

/** Варианты решения в порядке кнопок. */
export const DISPOSITIONS: readonly Disposition[] = ['rework', 'repair', 'use_as_is', 'scrap', 'return_to_supplier']

/** Ключ текста кнопки варианта решения. */
export const DISPOSITION_LABEL: Record<Disposition, string> = {
  rework: 'decisions.disposition.rework',
  repair: 'decisions.disposition.repair',
  use_as_is: 'decisions.disposition.useAsIs',
  scrap: 'decisions.disposition.scrap',
  return_to_supplier: 'decisions.disposition.returnToSupplier',
}

/** Операция действия. */
export const operationOf = (action: DecisionAction): string => (action === 'disposition' ? DISPOSITION_ACTION.operation : SIGNAL_ACTIONS[action].operation)

/** Ремонт и «как есть» — только по действующему разрешению на отклонение (FR-53). */
export const needsConcession = (d: Disposition | null | undefined): boolean => d === 'repair' || d === 'use_as_is'

/**
 * Действия по сигналу, которые имеет смысл показать: сигнал на рассмотрении или
 * ждёт доп. проверки. «Принять — передать» — только если сервер назвал
 * следующий шаг; «изолировать» — если изделие ещё не в изоляции.
 */
export function signalActionsFor(card: Pick<NcCard, 'signal_status' | 'next_step_label' | 'item_statuses'>): SignalAction[] {
  if (card.signal_status !== 'under_review' && card.signal_status !== 'recheck_assigned') return []
  return SIGNAL_ACTION_ORDER.filter((a) => {
    if (a === 'accept_and_pass') return !!card.next_step_label
    if (a === 'isolate') return card.item_statuses.position !== 'isolated'
    return true
  })
}

/** Открыто ли решение по несоответствию: подтверждено, решения ещё нет, не закрыто. */
export const dispositionOpen = (card: Pick<NcCard, 'signal_status' | 'item_statuses' | 'status_by_item'>): boolean =>
  card.signal_status === 'confirmed' && card.item_statuses.disposition === 'none' && card.status_by_item !== 'closed'

/**
 * Разрешения, которые можно выбрать: действует, покрывает изделие, лимит не
 * исчерпан. Решает применимость сервер; здесь — только показ выбора.
 */
export const usableConcessions = (list: readonly ConcessionOption[]): ConcessionOption[] =>
  list.filter((c) => c.status === 'active' && c.applies_to_item && c.used < c.limit)

/** Черновик решения в панели — то, что человек выбрал и написал. */
export interface DecisionDraft {
  action: DecisionAction
  /** Вариант решения для `disposition`. */
  disposition?: Disposition | null
  /** Выбранное разрешение на отклонение. */
  concession_id?: string | null
  /** Метод доп. проверки для `request_recheck`. */
  method?: InspectionMethod | null
  /** Причина или основание. */
  reason: string
}

/**
 * Что мешает подписать черновик — ключи текстов; пусто — можно к подписи.
 * @param draft — черновик
 * @param card — карточка
 * @param concessions — разрешения на отклонение по изделию
 */
export function draftProblems(draft: DecisionDraft, card: Pick<NcCard, 'item_processed'>, concessions: readonly ConcessionOption[]): string[] {
  const out: string[] = []
  const reasonEmpty = !draft.reason.trim()
  if (draft.action === 'reject_signal' && reasonEmpty) out.push('errors.decision.rejectReasonRequired')
  if (draft.action === 'request_recheck' && !draft.method) out.push('widgets.decisions.methodRequired')
  if (draft.action === 'disposition') {
    if (!draft.disposition) out.push('widgets.decisions.dispositionRequired')
    if (reasonEmpty) out.push('widgets.decisions.reasonRequired')
    if (needsConcession(draft.disposition)) {
      const ok = usableConcessions(concessions).some((c) => c.concession_id === draft.concession_id)
      if (!ok) out.push('decisions.disposition.selectConcession')
    }
    if (draft.disposition === 'return_to_supplier' && card.item_processed) out.push('decisions.disposition.returnOnlyUnprocessed')
  }
  return out
}

/** Команда решения для операции API — поля как в `data` записи журнала. */
export interface DecisionCommand {
  operation: string
  nc_id: string
  item_id: string
  signal_ids: string[]
  disposition?: Disposition
  concession_id?: string
  method?: InspectionMethod
  resolution?: 'accept'
  reason?: { text: string }
}

/**
 * Команда из черновика: только поля контракта соответствующей записи.
 * @param draft — проверенный черновик
 * @param card — карточка
 */
export function decisionCommand(draft: DecisionDraft, card: Pick<NcCard, 'nc_id' | 'item_id' | 'signal'>): DecisionCommand {
  const cmd: DecisionCommand = { operation: operationOf(draft.action), nc_id: card.nc_id, item_id: card.item_id, signal_ids: [card.signal.signal_id] }
  const reason = draft.reason.trim()
  if (reason) cmd.reason = { text: reason }
  if (draft.action === 'disposition' && draft.disposition) {
    cmd.disposition = draft.disposition
    if (needsConcession(draft.disposition) && draft.concession_id) cmd.concession_id = draft.concession_id
  }
  if (draft.action === 'request_recheck' && draft.method) cmd.method = draft.method
  if (draft.action === 'accept_and_pass') cmd.resolution = 'accept'
  return cmd
}


/**
 * Сводка решения для окна подписи уровня 2: действие, изделие, несоответствие,
 * вид дефекта и зона, разрешение на отклонение, причина. Не больше 7 полей.
 * @param draft — черновик
 * @param card — карточка
 * @param concessions — разрешения (для номера выбранного)
 */
export function decisionSummary(draft: DecisionDraft, card: NcCard, concessions: readonly ConcessionOption[] = []): SummaryField[] {
  const fields: SummaryField[] = []
  if (draft.action === 'disposition') {
    fields.push({ labelKey: 'widgets.signing.fields.action', valueKey: draft.disposition ? DISPOSITION_LABEL[draft.disposition] : 'common.words.unknown', valueParams: { operation: card.rework_operation_label ?? '' } })
  } else {
    // «Назначить доп. проверку» — с текстом «изделие ждёт в изоляции», метод — отдельным полем.
    const key = draft.action === 'request_recheck' ? 'decisions.signal.requestRecheckWaiting' : SIGNAL_ACTIONS[draft.action].labelKey
    fields.push({ labelKey: 'widgets.signing.fields.action', valueKey: key, valueParams: { nextStep: card.next_step_label ?? '' } })
  }
  fields.push({ labelKey: 'common.words.item', value: card.item_label })
  fields.push({ labelKey: 'common.words.nonconformity', value: card.number })
  const defect = card.signal.defect_type_label ?? card.signal.defect_type_code
  if (defect) fields.push({ labelKey: 'common.words.defectType', value: card.signal.zone_label ? `${defect} · ${card.signal.zone_label}` : defect })
  if (draft.action === 'request_recheck' && draft.method) fields.push({ labelKey: 'inspection.method.title', valueKey: METHOD_TEXT[draft.method] })
  if (draft.action === 'disposition' && needsConcession(draft.disposition)) {
    const c = concessions.find((x) => x.concession_id === draft.concession_id)
    if (c) fields.push({ labelKey: 'decisions.concession.title', value: c.number })
  }
  if (draft.reason.trim()) fields.push({ labelKey: 'common.words.basis', value: draft.reason.trim() })
  fields.push({ labelKey: 'widgets.signing.fields.level', valueKey: 'decisions.signature.level2' })
  return fields.slice(0, SUMMARY_MAX)
}
