/** Реестр процессов (эпик 39, UI-11, Д-70): список, «Создать процесс», окно процесса в адресе. */
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { computed, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { abilitiesPlugin } from '@casl/vue'
import { describe, expect, it, vi } from 'vitest'
import { createAppAbility } from '@/shared/lib/access'
import { i18n } from '@/shared/i18n'
import type { ProcessSummary } from '../model/source'
import Widget from '../ui/ProcessRegistryWidget.vue'

// Окно процесса тянет модельер bpmn-js — импорт под нагрузкой машины небыстрый.
vi.setConfig({ testTimeout: 60_000 })

const processes: ProcessSummary[] = [
  { process_id: 'Process_Flange', name: 'Фланец люка гермокорпуса в сборе', active_version: { version_id: 'flange-1', label: 'v1' }, versions: 2, status: 'active', is_default: true, items_in_work: 34 },
  { process_id: 'Process_Bracket', name: 'Кронштейн крепления приборной панели', versions: 1, status: 'draft', is_default: false, items_in_work: 0 },
]

vi.mock('../model/source', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../model/source')>()),
  useProcesses: () => ({
    query: { isLoading: ref(false), error: ref(null), refetch: vi.fn() },
    processes: computed(() => processes),
    mode: computed(() => 'live'),
  }),
}))

async function mountRegistry(actions: string[]) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
  await router.push('/')
  const ability = createAppAbility()
  ability.update(actions.map((action) => ({ action, subject: 'all' })))
  const w = mount(Widget, {
    props: { widgetId: 'process-registry', titleKey: 'processEditor.registry.title', slotId: 'processes', slice: {}, density: 'compact' },
    global: {
      plugins: [createPinia(), i18n, router, [abilitiesPlugin, ability]],
      stubs: { ProcessDrawer: { props: ['process', 'versionId'], template: '<div data-testid="drawer" :data-open="process?.process_id" />' }, CreateProcessDialog: true },
    },
  })
  await flushPromises()
  return { w, router }
}

describe('реестр процессов', () => {
  it('процессы с действующей версией, числом версий, состоянием; основной отмечен', async () => {
    const { w } = await mountRegistry(['process.version.draft'])
    const rows = w.findAll('tr[data-process]')
    expect(rows.map((r) => r.attributes('data-process'))).toEqual(['Process_Flange', 'Process_Bracket'])
    expect(rows[0]!.text()).toContain('основной')
    expect(rows[0]!.text()).toContain('v1')
    expect(rows[1]!.text()).toContain('Не введён')
    expect(w.find('[data-action="create"]').exists()).toBe(true)
  })

  it('щелчок по строке открывает окно процесса и пишет его в адрес', async () => {
    const { w, router } = await mountRegistry([])
    await w.find('tr[data-process="Process_Bracket"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.open).toBe('process:Process_Bracket')
    expect(w.find('[data-testid="drawer"]').attributes('data-open')).toBe('Process_Bracket')
    // Без права на черновик кнопки «Создать процесс» нет.
    expect(w.find('[data-action="create"]').exists()).toBe(false)
  })
})
