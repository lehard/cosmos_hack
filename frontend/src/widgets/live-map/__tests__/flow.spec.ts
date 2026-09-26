// «Карта производства» (режим руководителя): участки — дорожки BPMN, счётчики и изделия по участкам,
// изделия инцидента и «локализовано N из M», статус участка.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import type { LiveMap } from '@/shared/api/generated/model'
import { containment, sectionStates, sectionsOf } from '@/entities/live-map'
import ProductionFlow from '../ui/ProductionFlow.vue'

const bpmn = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:ant="urn:ant:bpmn-ext:1" id="D">
  <bpmn:process id="P">
    <bpmn:laneSet id="LS">
      <bpmn:lane id="L_M" name="Механический цех"><bpmn:flowNodeRef>M1</bpmn:flowNodeRef></bpmn:lane>
      <bpmn:lane id="L_W" name="Сварочный цех"><bpmn:flowNodeRef>W1</bpmn:flowNodeRef><bpmn:flowNodeRef>W2</bpmn:flowNodeRef></bpmn:lane>
      <bpmn:lane id="L_E" name="Пустая дорожка"><bpmn:flowNodeRef>G1</bpmn:flowNodeRef></bpmn:lane>
    </bpmn:laneSet>
    <bpmn:task id="M1" name="Мехобработка"><bpmn:extensionElements><ant:properties stepKey="machining.cnc" /></bpmn:extensionElements></bpmn:task>
    <bpmn:task id="W1" name="Сварка"><bpmn:extensionElements><ant:properties stepKey="welding.weld" /></bpmn:extensionElements></bpmn:task>
    <bpmn:task id="W2" name="КТ-3"><bpmn:extensionElements><ant:properties stepKey="welding.kt3" /></bpmn:extensionElements></bpmn:task>
    <bpmn:exclusiveGateway id="G1" />
  </bpmn:process>
</bpmn:definitions>`

const item = (id: string, step: string, position: string, incident_status?: string) =>
  ({ item_id: id, label: id, step_key: step, position, process_version_id: 'v1', summary: 'in_process', incident_status }) as never

const map = (): LiveMap =>
  ({
    bpmn_xml: bpmn,
    counters: [
      { step_key: 'machining.cnc', queue: 1, in_progress: 2, passed: 5, defects: 0 },
      { step_key: 'welding.weld', queue: 3, in_progress: 1, passed: 4, defects: 1 },
      { step_key: 'welding.kt3', queue: 2, in_progress: 0, passed: 3, defects: 0 },
    ],
    items: [item('Ф-001', 'machining.cnc', 'in_progress'), item('Ф-002', 'welding.weld', 'in_queue', 'suspect'), item('Ф-003', 'welding.kt3', 'isolated', 'confirmed'), item('Ф-004', 'welding.weld', 'in_progress', 'excluded')],
    anomalies: [{ step_key: 'welding.kt3', step_name: 'КТ-3', kind: 'queue_above_norm', threshold: 'норма 2' }],
    bottleneck: { step_key: 'welding.weld', step_name: 'Сварка', wait: '38 мин' },
    data_gaps: [],
    incident: { incident_id: 'RS-01', label: 'Инцидент ИС-2', size: 6, size_at_creation: 34, scope_version: 3 },
    process_id: 'P', process_name: 'Фланец', process_version: { version_id: 'v1', label: 'v1' }, versions: [], basis_seq: 1,
  }) as never

describe('карта производства', () => {
  it('участки — дорожки BPMN с шагами по stepKey; пустые дорожки пропущены', () => {
    const s = sectionsOf(bpmn)
    expect(s.map((x) => x.name)).toEqual(['Механический цех', 'Сварочный цех'])
    expect(s[1]!.stepKeys).toEqual(['welding.weld', 'welding.kt3'])
  })

  it('по участку: суммы счётчиков, изделия, изделия области риска и локализованные, статус', () => {
    const [mech, weld] = sectionStates(map(), sectionsOf(bpmn))
    expect(mech).toMatchObject({ queue: 1, inProgress: 2, passed: 5, status: 'normal' })
    expect(weld).toMatchObject({ queue: 5, inProgress: 1, passed: 7, defects: 1, isolated: 1, bottleneck: '38 мин' })
    expect(weld!.inScope.map((i) => i.item_id)).toEqual(['Ф-002', 'Ф-003'])
    expect(weld!.status).toBe('danger')
    expect(containment([mech!, weld!])).toEqual({ inScope: 2, isolated: 1 })
  })

  it('экран: инцидент с путём и «локализовано N из M», участки со статусом, лента событий до текущего момента', async () => {
    const w = mount(ProductionFlow, {
      props: {
        map: map(),
        scopePath: [34, 13, 6],
        marks: [
          { mark_id: '1', at: '2026-09-23T08:21:00Z', kind: 'spike', title: 'ИС-2 вышел за уставку' },
          { mark_id: '2', at: '2026-09-23T12:00:00Z', kind: 'escalation', title: 'из будущего' },
        ],
        now: Date.parse('2026-09-23T09:00:00Z'),
      },
      global: { plugins: [createPinia(), i18n] },
    })
    expect(w.find('[data-testid="flow-incident"]').text()).toContain('6 изделий в области риска')
    expect(w.find('[data-testid="flow-path"]').text()).toBe('34 → 13 → 6')
    expect(w.find('[data-testid="flow-contained"]').text()).toContain('локализовано 1 из 2')
    const weld = w.find('[data-section="L_W"]')
    expect(weld.attributes('data-status')).toBe('danger')
    expect(weld.find('[data-testid="section-scope"]').text()).toBe('в области риска: 2 · локализовано 1')
    // Под числом — одна строка о главном: изделия инцидента важнее узкого места.
    expect(weld.find('[data-testid="section-bottleneck"]').exists()).toBe(false)
    expect(w.find('[data-testid="flow-feed"]').text()).toContain('ИС-2 вышел за уставку')
    expect(w.find('[data-testid="flow-feed"]').text()).not.toContain('из будущего')
    await weld.find('.section-head').trigger('click')
    expect(w.emitted('open-section')).toHaveLength(1)
  })
})
