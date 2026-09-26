// Паспорт изделия: у наблюдения анализатора — «Как машина пришла к выводу»;
// дорожка раскрывается по кнопке и только тогда читает наблюдение и паспорт допуска.
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import type { PassportEntry } from '@/shared/api/generated/model'
import PassportEntries from '@/widgets/item-passport/ui/PassportEntries.vue'

let pinia: ReturnType<typeof createPinia>
beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
})
afterEach(() => vi.unstubAllGlobals())

const entry = (over: Partial<PassportEntry>): PassportEntry => ({
  event_id: 'E-1',
  event_type: 'process.operation.finished',
  kind: 'fact' as never,
  occurred_at: '2026-09-21T08:00:00Z',
  recorded_at: '2026-09-21T08:00:00Z',
  seq: 1,
  signatures: [],
  summary: 'Сварка завершена',
  ...over,
})

describe('паспорт изделия — дорожка решения у наблюдения', () => {
  it('кнопка только у наблюдения анализатора; по нажатию — дорожка из vision.observation.read', async () => {
    const calls = mockApi({
      'GET /api/v1/vision/observations/E-OBS-3': {
        event_id: 'E-OBS-3',
        occurred_at: '2026-09-21T08:30:00Z',
        point: 'КТ-3',
        outcome: 'no_defect_indicated',
        quality_bp: 3400,
        confidence_bp: 8800,
        level_then: 3,
        passport_id: 'AP-KT3-WELD-1',
        status_then: 'active',
        status_now: 'active',
        allowed_auto_actions: [],
        suspicious: true,
        reasons: ['Качество кадра ниже карты контроля'],
        versions: {},
      },
      'GET /api/v1/analyzer-passports/AP-KT3-WELD-1': { passport_id: 'AP-KT3-WELD-1', trust_level: 3, monitor: { window: 20, quality_min_bp: 6000, seen: 1, recent_quality_bp: [] } },
    })
    const entries = [entry({}), entry({ event_id: 'E-OBS-3', event_type: 'inspection.result.recorded', source_kind: 'camera', seq: 2, summary: 'КТ-3: признаков нет' })]
    const w = await mountWidget(PassportEntries, { entries }, pinia)
    const toggles = w.findAll('[data-testid="trace-toggle"]')
    expect(toggles).toHaveLength(1)
    expect(calls.some((c) => c.path.startsWith('/api/v1/vision') || c.path.startsWith('/api/v1/analyzer-passports'))).toBe(false)
    await toggles[0]!.trigger('click')
    await settle()
    expect(w.find('[data-testid="decision-trace"]').exists()).toBe(true)
    expect(w.find('[data-step="quality"]').attributes('data-below')).toBe('true')
    expect(w.find('[data-step="quality"]').text()).toContain('порог допуска анализатора 0,60')
    w.unmount()
  })
})
