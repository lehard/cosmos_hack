/**
 * Изоляция изделия и физическое перемещение в изолятор (FR-55, FR-137, UJ-7).
 *
 * «Изолировано в системе» ставит решение человека (nonconformity), «физически
 * перемещено» — только подтверждённая приёмка в изоляторе
 * (`process.movement.receive`, `destination_kind = isolator`). Пока приёмки нет,
 * карточка несоответствия несёт расхождение `physically_not_moved`; его и
 * показываем, ничего не вычисляя сами (NFR-UI-4).
 */
import type { NCCard, ReceiveMovement, ReceiveMovementInspectionOnReceipt } from '@/shared/api/generated/model'

/** Состояние изоляции для экрана. */
export type IsolationState =
  /** Изолировано в системе, физически не перемещено — нужна приёмка в изоляторе. */
  | 'not_moved'
  /** Перемещение в изолятор подтверждено. */
  | 'moved'
  /** Изделие не изолировано. */
  | 'none'
  /** Сведений нет: несоответствия не найдено или карточка не прочиталась. */
  | 'unknown'

/**
 * Состояние изоляции по карточке несоответствия изделия.
 * @param card — карточка любого несоответствия изделия (изоляция — на изделии)
 */
export function isolationStateOf(card: Pick<NCCard, 'isolation' | 'physically_not_moved'> | null | undefined): IsolationState {
  if (!card) return 'unknown'
  if (card.physically_not_moved) return 'not_moved'
  if (!card.isolation) return 'none'
  return card.isolation.physically_moved ? 'moved' : 'not_moved'
}

/** Варианты осмотра при приёмке (порядок — как на экране). */
export const INSPECTION_ON_RECEIPT: readonly ReceiveMovementInspectionOnReceipt[] = ['no_damage', 'damage_found', 'not_inspected']

/** Черновик подтверждения из формы. */
export interface IsolatorMoveDraft {
  to_location_id: string
  inspection_on_receipt: ReceiveMovementInspectionOnReceipt
}

/** Общие поля команды (AD-7, AD-39, барьер 2 AD-15). */
export interface CommandMeta {
  command_id: string
  basis_seq: number
  policy_seq: number
  workplace_id?: string
}

/**
 * Тело `process.movement.receive` для приёмки в изоляторе: вид места — изолятор.
 * @param draft — что выбрал человек
 * @param meta — id команды, seq объекта и политики, рабочее место сеанса
 */
export function isolatorReceiveBody(draft: IsolatorMoveDraft, meta: CommandMeta): ReceiveMovement {
  return {
    ...meta,
    destination_kind: 'isolator',
    to_location_id: draft.to_location_id.trim(),
    inspection_on_receipt: draft.inspection_on_receipt,
  }
}
