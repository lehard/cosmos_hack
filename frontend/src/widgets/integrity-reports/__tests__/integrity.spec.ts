// Верификатор на столе Аудитора ИБ (FR-106, FR-108; AD-46): состояние «по
// данным сервера», отчёты, проверки выбранного отчёта; только чтение.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { report, reportSummary } from '@/entities/integrity/__tests__/fixtures'
import IntegrityReportsWidget from '../ui/IntegrityReportsWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'integrity-reports', titleKey: 'audit.verifier.title', density: 'compact' }
const digest = 'sha256:0123456789abcdef0123'
const routes = (over: Record<string, unknown> = {}) => ({
  'GET /api/v1/integrity': { status: 'violated', server_side: true, interval_seconds: 300, checked_at: '2026-09-23T13:40:00Z' },
  'GET /api/v1/verifier-reports': { items: [reportSummary(), reportSummary({ report_digest: 'sha256:old', verdict: 'intact', checked_at: '2026-09-23T08:00:00Z' })] },
  [`GET /api/v1/verifier-reports/${digest}`]: report(),
  ...over,
})

describe('верификатор', () => {
  it('нарушение: «по данным сервера», признак на рамке, последний отчёт открыт с проверками', async () => {
    mockApi(routes())
    const w = await mountWidget(IntegrityReportsWidget, props)
    await settle()
    expect(w.find('.widget-frame').attributes('data-state')).toBe('defect_indication')
    expect(w.find('[data-testid="status"]').text()).toContain('Нарушение целостности')
    expect(w.find('[data-testid="status"]').text()).toContain('по данным сервера')
    const rows = w.findAll('li.row')
    expect(rows[0]!.attributes('aria-selected')).toBe('true')
    expect(rows[0]!.text()).toContain('Нарушено')
    expect(rows[0]!.text()).toContain('Проверено до записи № 229999')
    const card = w.find('[data-testid="report"]')
    expect(card.find('tr[data-check="цепочки"]').text()).toContain('Отвергнуто')
    expect(card.find('tr[data-check="цепочки"]').text()).toContain('CA-17')
    expect(card.find('tr[data-check="момент подписи"]').text()).toContain('Не проверяемо')
    expect(card.text()).toContain('по виртуальному времени')
    expect(card.text()).toContain('scenario: 2280')
    expect(card.text()).toContain('Бумажных решений для сверки с оригиналами: 1')
  })

  it('список отчётов не прочитан — ошибка, а не «отчётов не было»', async () => {
    mockApi(routes({ 'GET /api/v1/verifier-reports': { __status: 501, __body: { type: 'urn:ant:problem:api.not_implemented', title: 'нет', status: 501, code: 'api.not_implemented' } } }))
    const w = await mountWidget(IntegrityReportsWidget, props)
    expect(w.find('[data-testid="reports-error"]').exists()).toBe(true)
    expect(w.text()).not.toContain('Отчётов верификатора ещё не было')
    expect(w.find('[data-testid="status"]').exists()).toBe(true)
  })

  it('кнопок изменения нет — только чтение', async () => {
    mockApi(routes())
    const w = await mountWidget(IntegrityReportsWidget, props)
    expect(w.findAll('button')).toHaveLength(0)
  })
})
