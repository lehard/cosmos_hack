// Изоляция и физическое перемещение в изолятор (FR-55): состояние — только из
// карточки несоответствия; команда — приёмка с видом места «изолятор».
import { describe, expect, it } from 'vitest'
import { isolationCard } from '@/entities/workplace/__tests__/fixtures'
import { isolationStateOf, isolatorReceiveBody } from '../model/isolation'

describe('изоляция изделия', () => {
  it('расхождение «изолировано в системе, физически не перемещено» — из карточки', () => {
    expect(isolationStateOf(isolationCard(false))).toBe('not_moved')
    expect(isolationStateOf(isolationCard(true))).toBe('moved')
    expect(isolationStateOf({ isolation: undefined, physically_not_moved: false })).toBe('none')
    expect(isolationStateOf(null)).toBe('unknown')
  })

  it('тело приёмки в изоляторе: вид места, изолятор, осмотр, поля команды', () => {
    const body = isolatorReceiveBody(
      { to_location_id: ' ISO-WC ', inspection_on_receipt: 'damage_found' },
      { command_id: '0190c0de-0000-7000-8000-000000000000', basis_seq: 120, policy_seq: 3, workplace_id: 'WP-WELD-2' },
    )
    expect(body).toEqual({
      command_id: '0190c0de-0000-7000-8000-000000000000',
      basis_seq: 120,
      policy_seq: 3,
      workplace_id: 'WP-WELD-2',
      destination_kind: 'isolator',
      to_location_id: 'ISO-WC',
      inspection_on_receipt: 'damage_found',
    })
  })
})
