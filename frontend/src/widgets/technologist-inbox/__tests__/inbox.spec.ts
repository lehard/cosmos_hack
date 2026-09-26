// Рабочий стол технолога (UI-37): задачи из данных сервера, по важности; «перейти» — раздел стола.
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { buildTasks } from '../model/tasks'
import InboxView from '../ui/InboxView.vue'

const input = () => ({
  incidents: [
    { incident_id: 'RS-01', label: 'Инцидент ИС-2', status: 'open', size: 6, initial_size: 34, scope_version: 3, opened_at: '2026-09-23T08:10:00Z', stage: 'hypothesis', counts: { confirmed: 1, suspect: 4, unknown: 1, excluded: 28 }, next_step: 'Контрольный образец на ИС-2', nc_ids: [], close_blockers: [], common_factor: { factor: 'machine', value: 'IS-2', label: 'Сварочный источник ИС-2' } },
  ],
  groups: [{ group_key: 'g', defect_type: 'burn_through', operation: 'welding.weld', equipment: 'IS-2', nc_count: 3, investigation: 'hypothesis_only', last_found_at: '2026-09-23T10:10:00Z', defect_type_label: 'Прожог', operation_label: 'Сварка фланца', incident_id: 'RS-01' }],
  actions: { items: [], summary: { open: 1, overdue: 1, ineffective: 0, hanging_temporary: 0, evaluation_due: 0, recurring: 1 }, recurring: [{ defect_type: 'burn_through', step_key: 'welding.weld', count: 3, nc_ids: [], with_action: false }], memory: [], as_of: '', basis_seq: 0 },
  versions: [{ process: { process_id: 'P', name: 'Фланец', is_default: true, items_in_work: 47, status: 'active', versions: 2 }, version: { version_id: 'V2', label: 'v2', status: 'draft', created_at: '', items_in_work: 0 } }],
})

describe('рабочий стол технолога', () => {
  it('задачи: расследование, повтор без мер, меры, черновик процесса; сломанное — первым', () => {
    const tasks = buildTasks(input() as never)
    expect(tasks.map((x) => x.kind)).toEqual(['investigation', 'recurring', 'actions', 'version'])
    expect(tasks[0]!.query).toEqual({ incident: 'RS-01' })
    expect(tasks[1]!.data).toMatchObject({ defect: 'Прожог', step: 'Сварка фланца', count: 3 })
    expect(tasks[1]!.query).toEqual({ incident: 'RS-01' })
  })

  it('экран: сколько требует действий, расследование словами со стадией и «Дальше»; переход', async () => {
    const w = mount(InboxView, { props: { tasks: buildTasks(input() as never) }, global: { plugins: [createPinia(), i18n] } })
    expect(w.find('[data-testid="inbox-count"]').text()).toBe('4')
    const inv = w.find('[data-kind="investigation"]')
    expect(inv.text()).toContain('Инцидент ИС-2 · Сварочный источник ИС-2')
    expect(inv.text()).toContain('Проверка гипотез')
    expect(inv.find('[data-testid="inbox-next"]').text()).toBe('Дальше: Контрольный образец на ИС-2')
    expect(w.find('[data-kind="recurring"]').text()).toContain('Повторяется: Прожог — Сварка фланца')
    expect(w.find('[data-kind="version"]').text()).toContain('«Фланец» — версия v2')
    await inv.find('button').trigger('click')
    expect(w.emitted('go')?.[0]?.[0]).toMatchObject({ tab: 'investigation' })
  })

  it('ничего не требует — так и сказано', () => {
    const w = mount(InboxView, { props: { tasks: [] }, global: { plugins: [createPinia(), i18n] } })
    expect(w.text()).toContain('Сейчас ничего не требует ваших действий')
  })
})
