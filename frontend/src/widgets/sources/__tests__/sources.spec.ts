// Источники и интеграции (FR-30, FR-33): устройства и ключи, пропуски номеров,
// отключение с основанием, статус обмена с внешними системами.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { adminSession, mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { channels, sources } from '@/entities/quarantine/__tests__/fixtures'
import { useMomentStore } from '@/shared/model/moment'
import SourcesWidget from '../ui/SourcesWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'sources', titleKey: 'desks.sources' }
const receipt = { command_id: 'x', seq: 1, event_ids: ['e'], replayed: false }
const routes = () => ({
  'GET /api/v1/sources': { items: sources() },
  'GET /api/v1/erp/channels': { items: channels() },
  'GET /api/v1/auth/session': adminSession,
  'POST /api/v1/ops/sources/WS-2/disable': receipt,
  'POST /api/v1/ops/sources/CAM-9/enable': receipt,
})

describe('источники', () => {
  it('устройства: ключ, последний номер, пропуски; подозрение на потерю — не «норма»', async () => {
    mockApi(routes())
    const w = await mountWidget(SourcesWidget, props)
    expect(w.find('tr[data-source="WS-2"]').text()).toContain('dev-ws2@1')
    expect(w.find('tr[data-source="WS-2"]').text()).toContain('1240')
    expect(w.find('tr[data-source="GW-2"]').text()).toContain('Подозрение на потерю данных')
    expect(w.find('tr[data-source="GW-2"] .warn').text()).toBe('35')
    expect(w.find('tr[data-source="CAM-9"]').text()).toContain('ещё не было')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('defect_indication')
  })

  it('отключение — только с основанием; команда ops.source.disable', async () => {
    const calls = mockApi(routes())
    const w = await mountWidget(SourcesWidget, props)
    const row = w.find('tr[data-source="WS-2"]')
    await row.find('[data-testid="switch"]').trigger('click')
    expect(row.find('[data-testid="confirm"]').attributes('disabled')).toBeDefined()
    await row.find('input').setValue('Замена датчика')
    await row.find('form').trigger('submit')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.path).toBe('/api/v1/ops/sources/WS-2/disable')
    expect(post?.body).toMatchObject({ reason: { text: 'Замена датчика' }, policy_seq: 7 })
    expect(w.find('tr[data-source="WS-2"]').text()).toContain('Команда принята')
  })

  it('отключённый источник — «Включить»', async () => {
    mockApi(routes())
    const w = await mountWidget(SourcesWidget, props)
    expect(w.find('tr[data-source="CAM-9"] [data-testid="switch"]').text()).toBe('Включить')
  })

  it('статус обмена: система, эмулятор, последний обмен, очередь', async () => {
    mockApi(routes())
    const w = await mountWidget(SourcesWidget, props)
    expect(w.find('[data-channel="onec"]').text()).toContain('1С')
    expect(w.find('[data-channel="onec"]').text()).toContain('эмулятор кейса')
    expect(w.find('[data-channel="mes"]').text()).toContain('С перебоями')
    expect(w.find('[data-channel="mes"]').text()).toContain('ответ 503')
  })

  it('воспроизведение — команды выключены', async () => {
    mockApi(routes())
    const { createPinia, setActivePinia } = await import('pinia')
    const pinia = createPinia()
    setActivePinia(pinia)
    useMomentStore().travel('2026-09-23T08:00:00Z')
    const w = await mountWidget(SourcesWidget, props, pinia)
    expect(w.find('tr[data-source="WS-2"] [data-testid="switch"]').attributes('disabled')).toBeDefined()
  })
})
