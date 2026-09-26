/**
 * Решения контролёра и решения по несоответствию (FR-52, FR-53, FR-54, FR-146):
 * какие действия есть, какой операцией API (x-ant-action, AD-40) и каким типом
 * записи журнала каждое исполняется, что обязательно перед подписью, какое тело
 * команды уходит на сервер и что показать в сводке окна подписи уровня 2
 * (FR-66, AD-13).
 *
 * Какие решения допустимы по состоянию, говорит сервер (`to_decide.decisions`);
 * есть ли на них права — `access.permission.list` (FR-85). Здесь — только то,
 * без чего команду заведомо отклонит проверка: причина (`reason` обязателен в
 * контракте; для отклонения сигнала — гард `nonconformity.reject_reason_required`),
 * разрешение на отклонение для ремонта и «как есть» (`nonconformity.concession_required`).
 */
import { SUMMARY_MAX, type SummaryField } from '@/entities/document'
import { METHOD_TEXT } from './texts'
import type {
  Concession,
  ConfirmNonconformity,
  Disposition,
  DsseEnvelope,
  InspectionMethod,
  IsolateItem,
  NCCard,
  RejectSignal,
  RequestRecheck,
  SetDisposition,
} from './types'

/** Действие панели решений. */
export type DecisionAction = 'confirm_nc' | 'reject_signal' | 'request_recheck' | 'isolate' | 'accept_and_pass' | 'disposition'

/** Описание действия: операция API, объект прав, тип записи, текст кнопки «действие — направление». */
export interface DecisionActionDef {
  /** x-ant-action id операции — ключ прав и объяснения. */
  operation: string
  /** Вид объекта прав операции (`x-ant-action.subject`). */
  subject: 'item' | 'nonconformity'
  /** Тип записи журнала, которую пишет операция (каталог AD-40). */
  eventType: string
  /** Ключ текста кнопки. */
  labelKey: string
  /** Причина обязательна (контракт команды). */
  reasonRequired: boolean
}

/** Действия в порядке кнопок (FR-52, FR-53). Уровень подписи всех — 2 (PRD §11.16). */
export const DECISION_ACTIONS: Record<DecisionAction, DecisionActionDef> = {
  confirm_nc: {
    operation: 'nonconformity.nonconformity.confirm',
    subject: 'nonconformity',
    eventType: 'decision.nonconformity.confirmed',
    labelKey: 'decisions.signal.confirmNc',
    reasonRequired: true,
  },
  // FR-52: отклонение без причины невозможно; исходный сигнал не меняется — решение новой записью.
  reject_signal: {
    operation: 'nonconformity.signal.reject',
    subject: 'item',
    eventType: 'decision.signal.rejected',
    labelKey: 'decisions.signal.rejectSignal',
    reasonRequired: true,
  },
  // Изделие не маршрутизируется автоматически — ждёт в изоляции (FR-52).
  request_recheck: {
    operation: 'nonconformity.recheck.request',
    subject: 'item',
    eventType: 'decision.recheck.requested',
    labelKey: 'decisions.signal.requestRecheckWaiting',
    reasonRequired: true,
  },
  isolate: {
    operation: 'nonconformity.item.isolate',
    subject: 'item',
    eventType: 'decision.item.isolated',
    labelKey: 'decisions.signal.isolate',
    reasonRequired: true,
  },
  accept_and_pass: {
    operation: 'nonconformity.presentation.resolve',
    subject: 'item',
    eventType: 'decision.presentation.resolved',
    labelKey: 'decisions.signal.acceptAndPass',
    reasonRequired: false,
  },
  disposition: {
    operation: 'nonconformity.disposition.set',
    subject: 'nonconformity',
    eventType: 'decision.disposition.set',
    labelKey: 'decisions.disposition.title',
    reasonRequired: true,
  },
}

/** Порядок действий по сигналу. */
export const SIGNAL_ACTION_ORDER: readonly DecisionAction[] = ['confirm_nc', 'reject_signal', 'request_recheck', 'accept_and_pass', 'isolate']

const BY_OPERATION = new Map(Object.entries(DECISION_ACTIONS).map(([a, d]) => [d.operation, a as DecisionAction]))

/**
 * Действия, допустимые по состоянию карточки (`to_decide.decisions` — id
 * операций), в порядке кнопок. Неизвестные id отбрасываются.
 * @param card — карточка
 */
export function availableActions(card: Pick<NCCard, 'to_decide'>): DecisionAction[] {
  const set = new Set(card.to_decide.decisions.map((op) => BY_OPERATION.get(op)).filter((a): a is DecisionAction => !!a))
  return [...SIGNAL_ACTION_ORDER, 'disposition' as const].filter((a) => set.has(a))
}

/** Варианты решения в порядке кнопок (FR-53). */
export const DISPOSITIONS: readonly Disposition[] = ['rework', 'repair', 'use_as_is', 'scrap', 'return_to_supplier']

/** Ключ текста кнопки варианта решения. */
export const DISPOSITION_LABEL: Record<Disposition, string> = {
  rework: 'decisions.disposition.rework',
  repair: 'decisions.disposition.repair',
  use_as_is: 'decisions.disposition.useAsIs',
  scrap: 'decisions.disposition.scrap',
  return_to_supplier: 'decisions.disposition.returnToSupplier',
}

/** Ремонт и «как есть» — только по действующему разрешению на отклонение (FR-53, FR-54). */
export const needsConcession = (d: Disposition | null | undefined): d is 'repair' | 'use_as_is' => d === 'repair' || d === 'use_as_is'

/**
 * Разрешения, которые можно выбрать для варианта: действует, для этого вида
 * решения, лимит не исчерпан. Область действия проверяет сервер (гард).
 * @param list — разрешения по изделию
 * @param disposition — ремонт или «как есть»
 */
export function usableConcessions(list: readonly Concession[], disposition: Disposition | null | undefined): Concession[] {
  if (!needsConcession(disposition)) return []
  return list.filter((c) => c.status === 'active' && c.kind === disposition && (c.limit == null || c.used < c.limit))
}

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
  /** Основание претензии поставщику (для «вернуть поставщику»). */
  claim_basis?: string
}

/**
 * Что мешает подписать черновик — ключи текстов; пусто — можно к подписи.
 * @param draft — черновик
 * @param concessions — разрешения на отклонение по изделию
 */
export function draftProblems(draft: DecisionDraft, concessions: readonly Concession[]): string[] {
  const out: string[] = []
  if (DECISION_ACTIONS[draft.action].reasonRequired && !draft.reason.trim()) {
    out.push(draft.action === 'reject_signal' ? 'errors.decision.rejectReasonRequired' : 'widgets.decisions.reasonRequired')
  }
  if (draft.action === 'request_recheck' && !draft.method) out.push('widgets.decisions.methodRequired')
  if (draft.action === 'accept_and_pass') out.push('widgets.decisions.presentationContextMissing')
  if (draft.action === 'disposition') {
    if (!draft.disposition) out.push('widgets.decisions.dispositionRequired')
    if (needsConcession(draft.disposition) && !usableConcessions(concessions, draft.disposition).some((c) => c.concession_id === draft.concession_id)) {
      out.push('decisions.disposition.selectConcession')
    }
  }
  return out
}

/** Сведения команды от сеанса и клиента (AD-7, AD-39, AD-15). */
export interface CommandMeta {
  /** UUIDv7 клиента — один на решение. */
  command_id: string
  /** Версия политики сеанса. */
  policy_seq: number
  /** Рабочее место сеанса. */
  workplace_id?: string
  /** Подписанный пакет агента токена; нет — команда без подписи агента. */
  signature?: DsseEnvelope
}

/** Команда решения: операция, объект пути и тело по контракту. */
export type DecisionRequest =
  | { action: 'confirm_nc'; nc_id: string; item_id: string; body: ConfirmNonconformity }
  | { action: 'reject_signal'; nc_id: string; item_id: string; body: RejectSignal }
  | { action: 'request_recheck'; nc_id: string; item_id: string; body: RequestRecheck }
  | { action: 'isolate'; nc_id: string; item_id: string; body: IsolateItem }
  | { action: 'disposition'; nc_id: string; item_id: string; body: SetDisposition }

/**
 * Тело команды из черновика — только поля контракта. `basis_seq` — из
 * карточки (AD-39: сервер отклонит, если объект изменился после неё).
 * @param draft — проверенный черновик (draftProblems пуст)
 * @param card — карточка
 * @param meta — сведения команды
 */
export function buildDecisionRequest(draft: DecisionDraft, card: Pick<NCCard, 'nc_id' | 'item_id' | 'basis_seq' | 'evidence'>, meta: CommandMeta): DecisionRequest {
  const common = {
    basis_seq: card.basis_seq,
    command_id: meta.command_id,
    policy_seq: meta.policy_seq,
    reason: { text: draft.reason.trim() },
    ...(meta.workplace_id ? { workplace_id: meta.workplace_id } : {}),
    ...(meta.signature ? { signature: meta.signature } : {}),
  }
  const ids = { nc_id: card.nc_id, item_id: card.item_id }
  const signals = card.evidence.signals
  const primary = signals[0]
  switch (draft.action) {
    case 'confirm_nc':
      return {
        action: 'confirm_nc',
        ...ids,
        body: {
          ...common,
          signal_ids: signals.map((s) => s.signal_id),
          severity: primary?.severity ?? 'unknown',
          ...(primary?.defect_type_code ? { defect_type_code: primary.defect_type_code } : {}),
          ...(card.evidence.requirement?.kd_ref ? { requirement_ref: card.evidence.requirement.kd_ref } : {}),
        },
      }
    case 'reject_signal':
      return { action: 'reject_signal', ...ids, body: { ...common, signal_ids: signals.map((s) => s.signal_id) } }
    case 'request_recheck': {
      const zones = signals.map((s) => s.zone_id).filter((z): z is string => !!z)
      return { action: 'request_recheck', ...ids, body: { ...common, method: draft.method ?? 'other', ...(zones.length ? { zone_ids: zones } : {}) } }
    }
    case 'isolate':
      return { action: 'isolate', ...ids, body: { ...common } }
    case 'disposition':
      return {
        action: 'disposition',
        ...ids,
        body: {
          ...common,
          disposition: draft.disposition ?? 'rework',
          ...(needsConcession(draft.disposition) && draft.concession_id ? { concession_id: draft.concession_id } : {}),
          ...(draft.disposition === 'return_to_supplier' && draft.claim_basis?.trim() ? { claim_basis: draft.claim_basis.trim() } : {}),
        },
      }
    case 'accept_and_pass':
      // Контекста точки предъявления (закрывающая точка, результаты методов) в карточке нет —
      // draftProblems не пускает этот черновик к подписи.
      throw new Error('accept_and_pass: нет контекста точки предъявления')
  }
}

/**
 * Сводка решения для окна подписи уровня 2 (FR-66): действие, изделие,
 * несоответствие, вид дефекта и зона, метод или разрешение, основание, уровень.
 * Не больше 7 полей.
 * @param draft — черновик
 * @param card — карточка
 * @param concessions — разрешения (для названия выбранного)
 */
export function decisionSummary(draft: DecisionDraft, card: Pick<NCCard, 'item_label' | 'number' | 'evidence'>, concessions: readonly Concession[] = []): SummaryField[] {
  const fields: SummaryField[] = []
  if (draft.action === 'disposition') {
    fields.push({ labelKey: 'widgets.signing.fields.action', valueKey: draft.disposition ? DISPOSITION_LABEL[draft.disposition] : 'common.words.unknown', valueParams: { operation: '' } })
  } else {
    fields.push({ labelKey: 'widgets.signing.fields.action', valueKey: DECISION_ACTIONS[draft.action].labelKey, valueParams: { nextStep: '' } })
  }
  fields.push({ labelKey: 'common.words.item', value: card.item_label })
  fields.push({ labelKey: 'common.words.nonconformity', value: card.number })
  const signal = card.evidence.signals[0]
  // Названия из справочников, код — только если названия нет (UI-24).
  const defect = signal?.defect_type_label ?? signal?.defect_type_code
  const zone = signal?.zone_label ?? signal?.zone_id
  if (defect) fields.push({ labelKey: 'common.words.defectType', value: zone ? `${defect} · ${zone}` : defect })
  if (draft.action === 'request_recheck' && draft.method) fields.push({ labelKey: 'inspection.method.title', valueKey: METHOD_TEXT[draft.method] })
  if (draft.action === 'disposition' && needsConcession(draft.disposition)) {
    const c = concessions.find((x) => x.concession_id === draft.concession_id)
    if (c) fields.push({ labelKey: 'decisions.concession.title', value: c.title })
  }
  if (draft.reason.trim()) fields.push({ labelKey: 'common.words.basis', value: draft.reason.trim() })
  fields.push({ labelKey: 'widgets.signing.fields.level', valueKey: 'decisions.signature.level2' })
  return fields.slice(0, SUMMARY_MAX)
}
