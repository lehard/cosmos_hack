// Экран «Интеграции» (FR-157): список систем, «не установлена», состояние и очередь.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { integrations } from '@/entities/integration/__tests__/fixtures'
import { mockApi, mountWidget } from '@/entities/run/__tests__/api'
import IntegrationsWidget from '../ui/IntegrationsWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'integrations', titleKey: 'widgets.integrations.title' }

describe('интеграции', () => {
  it('системы: состояние, «не установлена», очередь / карантин; выключенная — не «норма»', async () => {
    mockApi({ 'GET /api/v1/ops/integrations': integrations() })
    const w = await mountWidget(IntegrationsWidget, props)
    expect(w.find('tr[data-system="onec"]').text()).toContain('1С:Предприятие')
    expect(w.find('tr[data-system="onec"]').text()).toContain('Стенд')
    expect(w.find('tr[data-system="onec"]').text()).toContain('2 / 1')
    expect(w.find('tr[data-system="galaktika"]').attributes('data-state')).toBe('not_installed')
    expect(w.find('tr[data-system="galaktika"]').text()).toContain('Не установлена')
    expect(w.find('tr[data-system="visionqc"]').text()).toContain('Выключена')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('defect_indication')
  })

  it('рабочий профиль — пояснение о запрете стенда', async () => {
    mockApi({ 'GET /api/v1/ops/integrations': integrations({ profile: 'prod' }) })
    const w = await mountWidget(IntegrationsWidget, props)
    expect(w.find('[data-testid="prod-no-stand"]').exists()).toBe(true)
  })
})
