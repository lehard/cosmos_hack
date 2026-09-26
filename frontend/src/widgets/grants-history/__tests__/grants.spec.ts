// История выдачи прав (FR-79; AD-11): что, кому, где, кем, вторая подпись,
// номер CA и документ; фильтр по сотруднику уходит на сервер.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, settle } from '@/entities/run/__tests__/api'
import { grants } from '@/entities/policy/__tests__/fixtures'
import GrantsHistoryWidget from '../ui/GrantsHistoryWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'grants-history', titleKey: 'widgets.grantsHistory', density: 'compact' }

describe('история выдачи прав', () => {
  it('записи: действие и вид по-русски, вторая подпись или её отсутствие, CA и документ', async () => {
    mockApi({ 'GET /api/v1/grants': { items: grants() } })
    const w = await mountWidget(GrantsHistoryWidget, props)
    const g1 = w.find('tr[data-seq="501"]')
    expect(g1.text()).toContain('Выдано: Роль «head_of_qc»')
    expect(g1.text()).toContain('P-PM-1')
    expect(g1.text()).toContain('CA-3')
    expect(g1.text()).toContain('Документ DOC-1')
    const g2 = w.find('tr[data-seq="502"]')
    expect(g2.text()).toContain('Отозвано: Клеймо «ST-5»')
    expect(g2.text()).toContain('без второй подписи')
  })

  it('фильтр по сотруднику — по Enter, в запрос', async () => {
    const calls = mockApi({ 'GET /api/v1/grants': { items: grants() } })
    const w = await mountWidget(GrantsHistoryWidget, props)
    const input = w.find('[data-testid="person"] input')
    await input.setValue('P-QC-3')
    await input.trigger('keydown', { key: 'Enter' })
    await settle()
    expect(calls.at(-1)?.query.get('person_id')).toBe('P-QC-3')
  })
})
