// Страница «Адаптация VisionQC» (эпик 40; FR-98…FR-101): паспорт, приостановленный
// автооткатом, открывается в правом окне; вернуть в работу может только начальник
// ОТК — иначе отказ с кодом; пропуск брака даёт изделия на перепроверку.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { adminSession, mockApi, mountWidget, problem, settle } from '@/entities/run/__tests__/api'
import VisionAdaptationWidget from '../ui/VisionAdaptationWidget.vue'

afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})
const props = { widgetId: 'vision-adaptation', titleKey: 'widgets.visionAdaptation.title' }
const versions = { recipe_ref: 'kt3-weld@1', analyzer_version: 'vqc-weld 2.3.1', camera_config: 'CAM-KT3/свет-R2', contract_version: '1.0' }
const passport = {
  passport_id: 'AP-KT3-WELD-1',
  analyzer_id: 'vqc-weld',
  title: 'Визуальный контроль шва (КТ-3)',
  stage: 'active',
  trust_level: 3,
  recipe_ref: 'kt3-weld@1',
  versions,
  status: 'suspended',
  admitted_at: '2026-01-10T08:00:00Z',
  document_id: 'DOC-1',
  allowed_auto_actions: ['record_observation'],
  basis_seq: 42,
  suspension: {
    event_id: '0190d0a4-0000-5000-8000-000000000001',
    trigger: 'drift',
    fallback: 'manual_control',
    at: '2026-09-26T10:00:00Z',
    note: 'дрейф входных данных: качество кадра ниже 0,70 в 3 наблюдениях подряд',
  },
  monitor: { window: 3, quality_min_bp: 7000, seen: 12, recent_quality_bp: [] },
}
const routes = (reinstate: unknown) => ({
  'GET /api/v1/analyzers': {
    items: [{ analyzer_id: 'vqc-weld', title: passport.title, kind: 'visionqc', passport_id: passport.passport_id, stage: 'active', trust_level: 3, status: 'suspended', versions }],
  },
  'GET /api/v1/analyzer-passports/AP-KT3-WELD-1': passport,
  'GET /api/v1/analyzer-passports/AP-KT3-WELD-1/checks': { items: [] },
  'GET /api/v1/vision/escapes': {
    items: [
      {
        event_id: 'e-1',
        defect_id: 'D-1',
        item_id: 'ENT01:F-240',
        analyzer_version: 'vqc-weld 2.3.1',
        method_covers_defect: true,
        recorded_at: '2026-09-26T11:00:00Z',
        missed: [{ event_id: 'o-1', item_id: 'ENT01:F-240', occurred_at: '2026-09-25T11:00:00Z', point: 'KT-3', versions }],
        recheck: [{ item_id: 'ENT01:F-238', observation_event_ids: ['o-2'], last_at: '2026-09-25T10:00:00Z' }],
      },
    ],
  },
  'GET /api/v1/vision/labeled-examples': { items: [] },
  'GET /api/v1/auth/session': adminSession,
  'POST /api/v1/analyzer-passports/AP-KT3-WELD-1/reinstate': reinstate,
})

describe('адаптация VisionQC', () => {
  it('приостановленный паспорт: триггер и пояснение в правом окне, возврат без начальника ОТК — отказ с кодом', async () => {
    const calls = mockApi(routes(problem(403, 'analyzer.reinstate_requires_head_of_qc')))
    const w = await mountWidget(VisionAdaptationWidget, props)
    const row = w.find('tr[data-passport="AP-KT3-WELD-1"]')
    expect(row.attributes('data-status')).toBe('suspended')
    await row.trigger('click')
    await settle()
    const drawer = document.body.querySelector('[data-testid="adaptation-drawer"]')
    expect(drawer?.textContent).toContain('Автооткат: дрейф входных данных')
    expect(drawer?.textContent).toContain('100 % ручной')
    ;(document.body.querySelector('[data-testid="reinstate"]') as HTMLButtonElement).click()
    await settle()
    const reason = document.body.querySelector('[data-testid="reason"] textarea') as HTMLTextAreaElement
    reason.value = 'Свет на посту восстановлен, эталонный набор пройден'
    reason.dispatchEvent(new Event('input'))
    await settle()
    ;(document.body.querySelector('[data-testid="submit"]') as HTMLButtonElement).click()
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.body).toMatchObject({ suspension_event_id: passport.suspension.event_id, basis_seq: 42, reason: { text: 'Свет на посту восстановлен, эталонный набор пройден' } })
    expect(document.body.querySelector('[data-testid="command-error"]')?.textContent).toBeTruthy()
  })

  it('пропуск брака: ранние «признаков нет» и изделия на перепроверку', async () => {
    mockApi(routes({ command_id: 'x', seq: 1, event_ids: ['x'], replayed: false }))
    const w = await mountWidget(VisionAdaptationWidget, props)
    await w.find('tr[data-escape="e-1"]').trigger('click')
    await settle()
    const recheck = document.body.querySelector('[data-testid="recheck"]')
    expect(recheck?.textContent).toContain('ENT01:F-238')
  })
})
