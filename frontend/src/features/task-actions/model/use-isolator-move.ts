/**
 * Подтвердить физическое перемещение в изолятор (FR-55, FR-137, UJ-7): мастер —
 * по задаче «перенести в изолятор», исполнитель — с терминала своего рабочего места.
 *
 * Состояние изоляции — из карточки несоответствия изделия (`nonconformity.card.read`;
 * изоляция — на изделии, поэтому годится карточка любого его несоответствия),
 * id несоответствий — из паспорта (`item.passport.read`). Изолятор по умолчанию —
 * из решения «изолировать», иначе изоляторы цеха рабочего места (справочник мест).
 * Команда — `process.movement.receive` с `destination_kind = isolator`.
 */
import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { usePassport } from '@/entities/item'
import { useNcCard } from '@/entities/nonconformity'
import { useOperationCommand } from '@/entities/operation'
import { isolatorsFor, useLocations, workshopOf, type RefLocation } from '@/entities/reference'
import { useSession } from '@/entities/session'
import { isolationStateOf, isolatorReceiveBody, type IsolationState, type IsolatorMoveDraft } from './isolation'

/** Подтверждение перемещения в изолятор для изделия. */
export function useIsolatorMove(itemId: MaybeRefOrGetter<string | null | undefined>) {
  const passport = usePassport(itemId)
  const ncId = computed(() => passport.data.value?.data.nonconformities.at(-1) ?? null)
  const card = useNcCard(ncId)
  const session = useSession()
  const locations = useLocations()
  const command = useOperationCommand()

  const state = computed<IsolationState>(() => {
    const p = passport.data.value?.data
    // Несоответствий у изделия нет — изоляции по решению нет.
    if (p && !p.nonconformities.length) return 'none'
    return isolationStateOf(card.data.value?.data)
  })

  /** Изолятор из решения «изолировать», если он указан. */
  const decidedIsolator = computed(() => card.data.value?.data.isolation?.isolator_location_id ?? null)

  /** Изоляторы для выбора: цеха рабочего места сеанса — первыми. */
  const isolators = computed<RefLocation[]>(() => {
    const all = locations.data.value?.data ?? []
    const wp = session.data.value?.data.workplace?.id
    return isolatorsFor(all, wp ? (workshopOf(all, wp)?.location_id ?? null) : null)
  })

  const itemLabel = computed(() => passport.data.value?.data.label ?? card.data.value?.data.item_label ?? toValue(itemId) ?? '')

  /**
   * Отправить подтверждение приёмки в изоляторе.
   * @param draft — изолятор и осмотр при приёмке
   * @param commandId — id команды: один на намерение, повтор — с тем же id (AD-7)
   */
  function confirm(draft: IsolatorMoveDraft, commandId: string) {
    const id = toValue(itemId)
    if (!id) return Promise.reject(new Error('изделие не выбрано'))
    const s = session.data.value?.data
    const basis = passport.data.value?.data.basis_seq ?? card.data.value?.data.basis_seq ?? 0
    const body = isolatorReceiveBody(draft, {
      command_id: commandId,
      basis_seq: basis,
      policy_seq: s?.policy_seq ?? 0,
      ...(s?.workplace?.id ? { workplace_id: s.workplace.id } : {}),
    })
    return command.mutateAsync({ kind: 'receive', item_id: id, body })
  }

  return {
    state,
    itemLabel,
    decidedIsolator,
    isolators,
    loading: computed(() => passport.isPending.value || (!!ncId.value && card.isPending.value)),
    readError: computed(() => (passport.error.value ?? (ncId.value ? card.error.value : null)) || null),
    busy: computed(() => command.isPending.value),
    commandError: computed(() => command.error.value ?? null),
    confirm,
  }
}
