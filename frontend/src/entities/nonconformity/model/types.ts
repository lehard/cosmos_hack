/**
 * Данные карточки несоответствия, очереди «Ждут моего решения», разрешений на
 * отклонение и команд решений (эпик 11; FR-51…FR-55, FR-146; кейс §2.3, §4.5) —
 * сгенерированные типы контракта (contracts/openapi.yaml → shared/api/generated):
 * `nonconformity.card.read`, `nonconformity.queue.list`,
 * `nonconformity.concession.list` и команды `nonconformity.*`.
 *
 * Карточка раздельно держит слои (FR-51): исходный сигнал (`evidence.signals`),
 * анализ системы — версиями (`system_analysis.versions`), решения людей
 * (`human_decisions`), итоговый статус — оси (`axes`). «Сигнал» ≠
 * «подтверждённое несоответствие» ≠ «предполагаемая причина» ≠
 * «подтверждённая ошибка». Необязательные поля в ответе отсутствуют вовсе —
 * читаются как `?? null`.
 */
import type { NCSourceSignalSeverity, RequestRecheckMethod, SetDispositionDisposition } from '@/shared/api/generated/model'

export type {
  Concession,
  ConcessionKind,
  ConcessionList,
  ConcessionStatus,
  ConfirmNonconformity,
  DecisionQueue,
  DecisionQueueRow,
  DecisionQueueRowKind,
  DsseEnvelope,
  IsolateItem,
  NCAnalyzerStage,
  NCCard,
  NCCardStatus,
  NCConclusionVersion,
  NCConclusionVersionOutcome,
  NCEvidence,
  NCHappened,
  NCItemAxes,
  NCOperationContext,
  NCReason,
  NCRecordRef,
  NCRecordRefKind,
  NCRequirement,
  NCSourceSignal,
  NCSourceSignalBasisKind,
  NCSystemAnalysis,
  NCToDecide,
  Receipt,
  RejectSignal,
  RequestRecheck,
  ResolvePresentation,
  SetDisposition,
} from '@/shared/api/generated/model'

/** Тяжесть дефекта по ГОСТ 15467. */
export type Severity = NCSourceSignalSeverity

/** Метод контроля. */
export type InspectionMethod = RequestRecheckMethod

/** Вариант решения по изделию (FR-53). */
export type Disposition = SetDispositionDisposition
