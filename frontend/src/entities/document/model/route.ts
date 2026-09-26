/**
 * Показ маршрута подписей и бумажной подписи (FR-66, FR-136, FR-139; AD-13,
 * AD-43). Засчитывает подписи и закрывает маршрут модуль documents — здесь только
 * счёт того, что сервер уже пометил `counted`, для строки «есть X из Y».
 */
import { QR_DOCUMENT_TEMPLATE } from '@/shared/contracts/constants'
import type { DocumentHead, RouteStage, StageSignature } from './types'

/** Прогресс маршрута для показа. */
export interface RouteProgress {
  /** Засчитанных подписей. */
  have: number
  /** Нужно всего. */
  need: number
  /** Этапы, где засчитанных подписей меньше нужного. */
  pending: RouteStage[]
  /** Подписи прежних версий документа — видны, но не засчитаны. */
  previous: StageSignature[]
}

/** Засчитанные подписи этапа. */
export const countedOf = (s: RouteStage): StageSignature[] => s.signatures.filter((x) => x.counted && !x.previous_version)

/**
 * Прогресс маршрута: засчитанные подписи против нужных по этапам.
 * @param route — этапы документа
 */
export function routeProgress(route: readonly RouteStage[]): RouteProgress {
  let have = 0
  let need = 0
  const pending: RouteStage[] = []
  const previous: StageSignature[] = []
  for (const stage of route) {
    const counted = Math.min(countedOf(stage).length, stage.required)
    have += counted
    need += stage.required
    if (counted < stage.required) pending.push(stage)
    previous.push(...stage.signatures.filter((x) => x.previous_version))
  }
  return { have, need, pending, previous }
}

/**
 * Содержимое QR бумажного экземпляра — `ant:doc:‹id›:‹отпечаток›`
 * (contracts/constants.yaml, соглашение «QR»). По нему сервер сверяет скан с
 * документом: чужой отпечаток не принимается (FR-139, `signing.qr_mismatch`).
 */
export const qrPayload = (doc: Pick<DocumentHead, 'document_id' | 'doc_digest'>): string =>
  QR_DOCUMENT_TEMPLATE.replace('{doc_id}', doc.document_id).replace('{doc_digest}', doc.doc_digest)

/**
 * Может ли пользователь заверить бумажную подпись (AD-43): заверитель ≠
 * подписант. Окончательно проверяет сервер (`signing.attester_is_signer`);
 * интерфейс только не предлагает заведомо отклоняемое.
 * @param attester — псевдоним того, кто загружает скан
 * @param signer — ожидаемый подписант этапа
 */
export const canAttest = (attester: string | null | undefined, signer: string | null | undefined): boolean =>
  !!attester && attester !== signer

/** Бумага разрешена на этапе (свойство маршрута шаблона `paper_allowed`). */
export const paperAllowed = (stage: Pick<RouteStage, 'paper_allowed'> | null | undefined): boolean => stage?.paper_allowed === true
