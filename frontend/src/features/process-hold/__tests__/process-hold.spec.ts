// «Остановить пост» / «Снять остановку» в окне поста (FR-49, Д-85): действующие
// остановки оборудования — с сервера (nonconformity.station.read), поэтому снять
// можно и после перезагрузки; нет полномочия — кнопка выключена и сказано почему.
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { adminSession, mockApi, mountWidget } from '@/entities/run/__tests__/api'
import type { StationView } from '@/shared/api/generated/model'
import ProcessHoldAction from '../ui/ProcessHoldAction.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
afterEach(() => vi.unstubAllGlobals())

const hold = {
  hold_id: 'HOLD-IS-2-0192aa01',
  reason: 'Ток ИС-2 176–182 А — вне уставки 160 ± 10',
  since: '2026-09-21T07:40:00Z',
  level: 'process_point_stop' as const,
  release_condition: 'Источник проверен и перекалиброван',
  equipment_id: 'IS-2',
}
const view = (over: Partial<StationView> = {}): StationView => ({
  step_key: 'IS-2',
  equipment_id: 'IS-2',
  basis_seq: 100,
  active_holds: [hold],
  actions: [
    { operation: 'nonconformity.process_hold.set', label: 'Остановить', allowed: false, why_available: 'Точка уже остановлена (HOLD-IS-2-0192aa01)', consequences: [] },
    {
      operation: 'nonconformity.process_hold.release',
      label: 'Снять',
      hold_id: 'HOLD-IS-2-0192aa01',
      allowed: true,
      why_available: 'У вас полномочие «Остановка и снятие остановки точки процесса»',
      consequences: ['Первые изделия после снятия — на усиленный контроль'],
    },
  ],
  ...over,
})
const perms = (actions: string[]) => ({ items: actions.map((a) => ({ action: a, action_class: 'decision', subject: 'equipment', object_id: 'IS-2' })) })
const props = { equipmentId: 'IS-2', equipmentTitle: 'Сварочный источник ИС-2' }

describe('остановка поста — окно участка', () => {
  it('после перезагрузки: действующая остановка с сервера и «Снять остановку» по ней', async () => {
    const calls = mockApi({
      'GET /api/v1/process/steps/IS-2/station-view': view(),
      'GET /api/v1/permissions': perms(['nonconformity.process_hold.set', 'nonconformity.process_hold.release']),
      'GET /api/v1/auth/session': adminSession,
      'GET /api/v1/journal/head': { seq: 100 },
    })
    const w = await mountWidget(ProcessHoldAction, props, pinia)
    expect(calls.find((c) => c.path === '/api/v1/process/steps/IS-2/station-view')?.query.get('equipment_id')).toBe('IS-2')
    const row = w.find('[data-hold="HOLD-IS-2-0192aa01"]')
    expect(row.text()).toContain('Остановлен с 21.09.2026, 10:40 · Стоп точки процесса')
    expect(row.text()).toContain('Основание: Ток ИС-2 176–182 А — вне уставки 160 ± 10')
    expect(row.text()).toContain('Условие снятия: Источник проверен и перекалиброван')
    // Пост уже остановлен — второй остановки не предлагаем.
    expect(w.find('[data-action="set-process-hold"]').exists()).toBe(false)
    const release = w.find('[data-action="release-process-hold"]')
    expect(release.attributes('disabled')).toBeUndefined()
    await release.trigger('click')
    expect(w.find('[data-testid="hold-consequences"]').text()).toContain('Первые изделия после снятия — на усиленный контроль')
    w.unmount()
  })

  it('нет полномочия снять — кнопка выключена и сказано почему', async () => {
    const v = view()
    v.actions[1] = { ...v.actions[1]!, allowed: false, why_available: 'Снять остановку может обладатель полномочия (начальник цеха, руководитель производства)' }
    mockApi({
      'GET /api/v1/process/steps/IS-2/station-view': v,
      'GET /api/v1/permissions': perms(['nonconformity.process_hold.set']),
      'GET /api/v1/auth/session': adminSession,
      'GET /api/v1/journal/head': { seq: 100 },
    })
    const w = await mountWidget(ProcessHoldAction, props, pinia)
    expect(w.find('[data-action="release-process-hold"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="release-why"]').text()).toContain('начальник цеха, руководитель производства')
    w.unmount()
  })

  it('остановок нет — «Остановить пост» с последствиями сервера', async () => {
    mockApi({
      'GET /api/v1/process/steps/IS-2/station-view': view({
        active_holds: [],
        actions: [{ operation: 'nonconformity.process_hold.set', label: 'Остановить', allowed: true, why_available: 'Точка работает', consequences: ['Новые изделия на шаг не пойдут'] }],
      }),
      'GET /api/v1/permissions': perms(['nonconformity.process_hold.set']),
      'GET /api/v1/auth/session': adminSession,
      'GET /api/v1/journal/head': { seq: 100 },
    })
    const w = await mountWidget(ProcessHoldAction, props, pinia)
    expect(w.find('[data-testid="active-holds"]').exists()).toBe(false)
    await w.find('[data-action="set-process-hold"]').trigger('click')
    expect(w.find('[data-testid="hold-consequences"]').text()).toContain('Новые изделия на шаг не пойдут')
    w.unmount()
  })

  it('окно участка не отвечает — как раньше: «Остановить пост» по праву', async () => {
    mockApi({
      'GET /api/v1/permissions': perms(['nonconformity.process_hold.set']),
      'GET /api/v1/auth/session': adminSession,
      'GET /api/v1/journal/head': { seq: 100 },
    })
    const w = await mountWidget(ProcessHoldAction, props, pinia)
    expect(w.find('[data-action="set-process-hold"]').exists()).toBe(true)
    expect(w.find('[data-action="release-process-hold"]').exists()).toBe(false)
    w.unmount()
  })
})
