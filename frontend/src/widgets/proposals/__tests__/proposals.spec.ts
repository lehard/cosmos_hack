// Каркас «Предложения» (FR-63): список пуст и так и сказано; генераторы не
// подключены; ограничение линии — основание, а не предложение.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget } from '@/entities/incident/__tests__/api-mock'
import ProposalsWidget from '../ui/ProposalsWidget.vue'

const nodes = (bottleneck?: unknown) => ({ period: { kind: 'shift', from: '2026-09-23T05:00:00.000Z', to: '2026-09-23T09:30:00.000Z' }, process_version_id: 'PV-1', counters: [], anomalies: [], data_gaps: [], basis_seq: 1, ...(bottleneck ? { bottleneck } : {}) })

describe('каркас «Предложения»', () => {
  afterEach(() => vi.unstubAllGlobals())
  const props = { widgetId: 'proposals', titleKey: 'desks.proposals' }

  it('ничего не применяется само; предложений нет; четыре генератора не подключены', async () => {
    mockApi({ 'GET /api/v1/metrics/node-counters': nodes({ step_key: 'quality.zt3', wait: '37 мин' }) })
    const w = await mountWidget(ProposalsWidget, props)
    expect(w.find('[data-testid="principle"]').text()).toContain('Система ничего не меняет сама')
    expect(w.find('[data-testid="no-proposals"]').text()).toBe('Предложений нет')
    expect(w.findAll('[data-generator]')).toHaveLength(4)
    expect(w.find('[data-testid="bottleneck"]').text()).toContain('quality.zt3')
    expect(w.find('[data-testid="bottleneck"]').text()).toContain('среднее ожидание 37 мин')
  })

  it('ограничения нет — так и сказано', async () => {
    mockApi({ 'GET /api/v1/metrics/node-counters': nodes() })
    const w = await mountWidget(ProposalsWidget, props)
    expect(w.find('[data-testid="no-bottleneck"]').exists()).toBe(true)
  })
})
