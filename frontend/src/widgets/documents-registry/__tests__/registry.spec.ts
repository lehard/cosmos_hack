// Реестр документов (FR-65, FR-66, FR-139; AD-12, AD-43; Д-70, Д-72): список
// с отбором на сервере, счётчики состояний, окно документа — маршрут подписей
// с классом ключа, отказы, кнопки по правам и отказ с замечанием.
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { abilitiesPlugin } from '@casl/vue'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { mockApi, settle } from '@/entities/run/__tests__/api'
import type { DocumentSummary, DocumentView } from '@/shared/api/generated/model'
import { i18n } from '@/shared/i18n'
import { createAppAbility } from '@/shared/lib/access'
import { RECORD_DRAWER } from '@/shared/model/record'
import { countByState, docState, filterParams, itemLabel, itemsOf, kindKey, keyStorageKey, templatesOf } from '../model/registry'
import DocumentDrawer from '../ui/DocumentDrawer.vue'
import DocumentsRegistryWidget from '../ui/DocumentsRegistryWidget.vue'

vi.setConfig({ testTimeout: 30_000 })
afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

/** Образцы — только для тестов. */
const rows = (): DocumentSummary[] => [
  {
    document_id: 'DOC-DISP-NC-G1', version: 1, template: 'nc-disposition@1', title: 'Решение комиссии по НС-И1', subject: { entity: 'nonconformity', id: 'NC-G1' },
    status: 'signing', state: 'signing', subject_label: 'НС-И1', item_ids: ['ENT01:F-015', 'ENT01:F-017'], process_id: 'Process_Flange',
    awaiting: { stage: 4, title: 'Представитель заказчика', candidates: ['CR-71'] }, stages_done: 3, stages_total: 4, updated_at: '2026-09-23T12:22:00Z',
  },
  {
    document_id: 'DOC-ACT-NC-06', version: 1, template: 'scrap-act@1', title: 'Акт о браке Ф-021', subject: { entity: 'nonconformity', id: 'NC-06' },
    status: 'signing', paper_status: 'printed', subject_label: 'НС-06', item_ids: ['ENT01:F-021'], stages_done: 1, stages_total: 3,
  },
  {
    document_id: 'DOC-PVA-FLANGE-1', version: 1, template: 'process-version-approval@1', title: 'Лист утверждения версии процесса', subject: { entity: 'process_version', id: 'flange-1' },
    status: 'route_closed', state: 'signed', subject_label: 'Фланец, версия v1', stages_done: 3, stages_total: 3,
  },
  { document_id: 'DOC-CONC-NC-04', version: 1, template: 'concession@1', title: 'Разрешение на отклонение', subject: { entity: 'nonconformity', id: 'NC-04' }, status: 'returned', item_ids: ['ENT01:F-019'] },
]

const view = (): DocumentView => ({
  document_id: 'DOC-ACT-NC-06', version: 1, template: 'scrap-act@1', doc_format_version: 1, title: 'Акт о браке Ф-021', subject: { entity: 'nonconformity', id: 'NC-06' },
  subject_label: 'НС-06', status: 'signing', content: { title: 'Акт о браке Ф-021' }, rendering_hash: 'streebog256:aa', doc_digest: 'streebog256:bb',
  summary_fields: [{ key: 'f1', label: 'Несоответствие', value: 'НС-06' }], source_event_ids: [], qr: 'ant:doc:DOC-ACT-NC-06:streebog256:bb', basis_seq: 250000,
  stages: [
    { stage: 1, authority: 'qc_acceptance', quorum: 'one', required: 1, level: 2, paper_allowed: true, candidates: ['INS-01'], signed_by: ['INS-01'], status: 'done' },
    { stage: 2, authority: 'nc_disposition', quorum: 'one', required: 1, level: 2, paper_allowed: true, candidates: ['HQC-01'], signed_by: [], status: 'pending' },
  ],
  signatures: [],
  route: [
    {
      stage: 1, title: 'Контролёр ОТК', authority_id: 'qc_acceptance', authority_label: 'Контролёр ОТК', quorum: 'one', required: 1, signature_level: 2, paper_allowed: true, done: true,
      signatures: [{ event_id: 'EV-1', signer_id: 'INS-01', class: 'personal', level: 2, check: 'valid', method: 'token_agent', signed_at: '2026-09-24T12:40:00Z', counted: true, key_ref: 'key-ins-01@1', key_storage: 'software_browser' }],
    },
    { stage: 2, title: 'Начальник ОТК', authority_id: 'nc_disposition', authority_label: 'Начальник ОТК', quorum: 'one', required: 1, signature_level: 2, paper_allowed: true, done: false, signatures: [] },
  ],
  versions: [{ version: 1, doc_digest: 'streebog256:bb', status: 'signing', drafted_at: '2026-09-24T12:35:00Z', signatures: 1 }],
  paper: [{ event_id: 'EV-P', version: 1, paper_status: 'printed', copy_no: '1', at: '2026-09-24T13:00:00Z' }],
})

const session = (id: string) => ({ user: { id, display_name: id }, policy_seq: 5, demo: true, active_role: 'head_of_qc', roles: [] })

describe('реестр документов: чистые функции', () => {
  it('состояние: сервер, запасной расчёт, «на бумаге» раньше «на подписи»', () => {
    expect(docState({ state: 'signed', status: 'route_closed' })).toBe('signed')
    expect(docState({ status: 'signing', paper_status: 'printed' })).toBe('paper')
    expect(docState({ status: 'route_closed', paper_status: 'printed' })).toBe('signed')
    expect(docState({ status: 'live' })).toBe('draft')
    expect(countByState(rows())).toMatchObject({ signing: 1, paper: 1, signed: 1, returned: 1, draft: 0 })
  })

  it('отбор без пустых полей; вид и изделие — для людей', () => {
    expect(filterParams({ process_id: 'Process_Flange', q: '  ', state: 'paper' })).toEqual({ process_id: 'Process_Flange', state: 'paper' })
    expect(kindKey('nc-disposition@1')).toBe('docRegistry.kinds.ncDisposition')
    expect(templatesOf(rows())).toEqual(['nc-disposition', 'scrap-act', 'process-version-approval', 'concession'])
    expect(itemLabel('ENT01:F-017')).toBe('Ф-017')
    expect(itemsOf(rows()).map((i) => i.label)).toEqual(['Ф-015', 'Ф-017', 'Ф-019', 'Ф-021'])
    expect(keyStorageKey('hardware_token')).toBe('docRegistry.key.hardwareToken')
    expect(keyStorageKey(undefined)).toBeNull()
  })
})

async function mountAt(component: object, props: Record<string, unknown>, actions: string[] = []) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
  await router.push('/')
  const ability = createAppAbility()
  ability.update(actions.map((action) => ({ action, subject: 'all' })))
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const w = mount(component, {
    props,
    attachTo: document.body,
    global: {
      plugins: [createPinia(), i18n, router, [VueQueryPlugin, { queryClient }], [abilitiesPlugin, ability]],
      provide: { [RECORD_DRAWER as symbol]: { kinds: new Set(['document']) } },
    },
  })
  await settle()
  return { w, router }
}

describe('раздел «Документы»', () => {
  it('строки: вид, номер, объект, состояние с прогрессом, кто подписывает; счётчики состояний', async () => {
    mockApi({ 'GET /api/v1/documents': { items: rows() }, 'GET /api/v1/processes': { items: [] } })
    const { w } = await mountAt(DocumentsRegistryWidget, { widgetId: 'documents-registry', titleKey: 'docRegistry.title', slotId: 's', slice: {}, density: 'compact' })
    const trs = w.findAll('tr[data-document]')
    expect(trs.map((r) => r.attributes('data-document'))).toEqual(['DOC-DISP-NC-G1', 'DOC-ACT-NC-06', 'DOC-PVA-FLANGE-1', 'DOC-CONC-NC-04'])
    expect(trs[0]!.text()).toContain('Решение по НП')
    expect(trs[0]!.text()).toContain('НС-И1')
    expect(trs[0]!.text()).toContain('На подписи')
    expect(trs[0]!.text()).toContain('этапов 3 из 4')
    expect(trs[0]!.text()).toContain('Представитель заказчика')
    expect(trs[0]!.text()).toContain('CR-71')
    expect(trs[1]!.attributes('data-state')).toBe('paper')
    expect(trs[3]!.text()).toContain('Возвращён')
    expect(w.find('[data-state="paper"] .count').text()).toBe('1')
    w.unmount()
  })

  it('отбор по состоянию уходит на сервер; щелчок по строке открывает окно документа', async () => {
    const calls = mockApi({ 'GET /api/v1/documents': { items: rows() }, 'GET /api/v1/processes': { items: [] } })
    const { w, router } = await mountAt(DocumentsRegistryWidget, { widgetId: 'documents-registry', titleKey: 'docRegistry.title', slotId: 's', slice: {}, density: 'compact' })
    await w.find('button[data-state="returned"]').trigger('click')
    await settle()
    expect(calls.some((c) => c.path === '/api/v1/documents' && c.query.get('state') === 'returned')).toBe(true)
    await w.find('tr[data-document="DOC-ACT-NC-06"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.open).toBe('document:DOC-ACT-NC-06')
    w.unmount()
  })
})

describe('окно документа', () => {
  const $ = (sel: string) => document.querySelector<HTMLElement>(sel)

  it('маршрут: кто подписал, каким ключом; начальник ОТК видит «Подписать» и «Отказать»', async () => {
    mockApi({
      'GET /api/v1/documents/DOC-ACT-NC-06': view(),
      'GET /api/v1/documents/DOC-ACT-NC-06/rendering': { document_id: 'DOC-ACT-NC-06', version: 1, rendering_hash: 'streebog256:aa', html: '<article>акт</article>' },
      'GET /api/v1/auth/session': session('HQC-01'),
    })
    const { w } = await mountAt(DocumentDrawer, { id: 'DOC-ACT-NC-06', show: true }, ['documents.document.sign', 'documents.signature.decline', 'documents.paper.print'])
    await vi.waitFor(() => expect($('[data-testid="doc-route"]')).not.toBeNull(), { timeout: 10_000 })
    const route = $('[data-testid="doc-route"]')!.textContent!
    expect(route).toContain('INS-01')
    expect(route).toContain('ключ в браузере')
    expect(route).toContain('ждёт подписи')
    expect(route).toContain('Может подписать: HQC-01')
    expect($('[data-action="sign"]')).not.toBeNull()
    expect($('[data-action="decline"]')).not.toBeNull()
    expect($('[data-action="print"]')).not.toBeNull()
    expect($('[data-testid="record-drawer-head"]')!.textContent).toContain('На бумаге')
    w.unmount()
  })

  it('без права подписи и не в маршруте — только печать и скачивание', async () => {
    mockApi({ 'GET /api/v1/documents/DOC-ACT-NC-06': view(), 'GET /api/v1/auth/session': session('AUD-01') })
    const { w } = await mountAt(DocumentDrawer, { id: 'DOC-ACT-NC-06', show: true }, [])
    await vi.waitFor(() => expect($('[data-action="download"]')).not.toBeNull(), { timeout: 10_000 })
    expect($('[data-action="sign"]')).toBeNull()
    expect($('[data-action="decline"]')).toBeNull()
    w.unmount()
  })

  it('отказ — только с замечанием, уходит documents.signature.decline с этапом', async () => {
    const calls = mockApi({
      'GET /api/v1/documents/DOC-ACT-NC-06': view(),
      'GET /api/v1/auth/session': session('HQC-01'),
      'POST /api/v1/documents/DOC-ACT-NC-06/route/declines': { command_id: 'c', seq: 1, event_ids: ['e'], replayed: false },
    })
    const { w } = await mountAt(DocumentDrawer, { id: 'DOC-ACT-NC-06', show: true }, ['documents.document.sign', 'documents.signature.decline'])
    await vi.waitFor(() => expect($('[data-action="decline"]')).not.toBeNull(), { timeout: 10_000 })
    $('[data-action="decline"]')!.click()
    await settle()
    const ta = document.querySelector<HTMLTextAreaElement>('[data-testid="doc-decline-form"] textarea')!
    ta.value = 'Нет подписи контролёра на бирке'
    ta.dispatchEvent(new Event('input'))
    await settle()
    $('[data-action="decline-send"]')!.click()
    await vi.waitFor(() => expect($('[data-testid="doc-done"]')).not.toBeNull(), { timeout: 10_000 })
    const post = calls.find((c) => c.method === 'POST')!
    expect(post.path).toBe('/api/v1/documents/DOC-ACT-NC-06/route/declines')
    expect(post.body).toMatchObject({ stage: 2, version: 1, comment: 'Нет подписи контролёра на бирке', doc_digest: 'streebog256:bb', policy_seq: 5 })
    w.unmount()
  })
})
