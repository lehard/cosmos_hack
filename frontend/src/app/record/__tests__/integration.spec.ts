// Окно интеграции (Д-70; FR-157, AD-47, Д-71): состояние, ошибки, карантин
// исходящих с переотправкой; «Выключить» — только с основанием, команда
// ops.integration.set; отказ гарда (стенд в prod) — текст ошибки в окне.
import { defineComponent, h, provide } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { integrations } from '@/entities/integration/__tests__/fixtures'
import { adminSession } from '@/entities/run/__tests__/api'
import { i18n } from '@/shared/i18n'
import { RECORD_DRAWER } from '@/shared/model/record'
import RecordDrawerHost from '../RecordDrawerHost.vue'
import { RECORD_KINDS, recordKinds } from '../registry'

const $ = (sel: string) => document.querySelector<HTMLElement>(sel)
const $$ = (sel: string) => [...document.querySelectorAll<HTMLElement>(sel)]
const until = (check: () => void) => vi.waitFor(check, { timeout: 10_000 })
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': status < 400 ? 'application/json' : 'application/problem+json', 'Ant-Backend': 'fixtures' } })
const receipt = { command_id: 'x', seq: 9, event_ids: ['e'], replayed: false }
const messages = {
  items: [
    {
      business_key: 'FL-0041/scrap/kt3', external_system: 'onec', action: 'scrap_transfer_rework', message_version: 1, status: 'rejected', after_rework: false,
      request_event_id: '0190a3f0-0000-7000-8000-000000000001', requested_at: '2026-09-26T07:00:00Z', basis_seq: 12,
      attempts: [{ at: '2026-09-26T07:00:00Z', outcome: 'rejected', error_code: 'E-422', error_message: 'не найден договор' }],
    },
    {
      business_key: 'FL-0042/accept', external_system: 'onec', action: 'accept_into_work', message_version: 1, status: 'acknowledged', after_rework: false,
      request_event_id: '0190a3f0-0000-7000-8000-000000000002', requested_at: '2026-09-26T07:00:00Z', basis_seq: 13, attempts: [],
    },
  ],
}

interface Call { method: string; path: string; body: unknown }
function serve(setStatus = 200): Call[] {
  const calls: Call[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string, init?: RequestInit) => {
      const path = decodeURIComponent(String(url).split('?')[0]!)
      const method = init?.method ?? 'GET'
      calls.push({ method, path, body: init?.body ? JSON.parse(String(init.body)) : undefined })
      if (path === '/api/v1/ops/integrations') return json(integrations())
      if (path === '/api/v1/auth/session') return json(adminSession)
      if (path === '/api/v1/erp/messages') return json(messages)
      if (method === 'POST' && path.endsWith('/state')) {
        return setStatus === 200 ? json(receipt) : json({ type: 'urn:ant:problem:ops.stand_forbidden', title: 'Стенд в рабочем профиле запрещён', status: 409, code: 'ops.stand_forbidden', detail: 'Профиль prod: стенд нельзя' }, 409)
      }
      if (method === 'POST') return json(receipt)
      return json({ type: 'urn:ant:problem:api.not_found', title: 'нет', status: 404, code: 'api.not_found' }, 404)
    }),
  )
  return calls
}

beforeAll(async () => {
  await recordKinds.integration!.load()
}, 60_000)
afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

const Shell = defineComponent({
  setup() {
    provide(RECORD_DRAWER, { kinds: RECORD_KINDS })
    return () => h(RecordDrawerHost)
  },
})

async function mountAt(path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/desk', component: { template: '<div />' } }] })
  await router.push(path)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const w = mount(Shell, { attachTo: document.body, global: { plugins: [createPinia(), i18n, router, [VueQueryPlugin, { queryClient }]] } })
  await flushPromises()
  return w
}

describe('окно интеграции', () => {
  it('1С: стенд, очередь, ошибка, карантин с переотправкой', async () => {
    const calls = serve()
    const w = await mountAt('/desk?open=integration:onec')
    await until(() => expect($$('[data-testid="quarantined-message"]').length).toBe(1))
    expect($('[data-testid="record-drawer-head"]')!.textContent).toContain('1С:Предприятие')
    expect($('[data-testid="integration-state"]')!.textContent).toContain('Стенд')
    expect($('[data-testid="integration-errors"]')!.textContent).toContain('не найден договор')
    // Реальный адрес 1С не задан — переключения нет, только «Выключить».
    expect($('[data-testid="integration-toggle"]')).toBeNull()
    $('[data-testid="resend"]')!.click()
    await until(() => expect(calls.some((c) => c.path === '/api/v1/erp/messages/FL-0041/scrap/kt3/resend')).toBe(true))
    w.unmount()
  })

  it('«Выключить» — только с основанием; команда ops.integration.set', async () => {
    const calls = serve()
    const w = await mountAt('/desk?open=integration:onec')
    await until(() => expect($('[data-testid="integration-disable"]')).not.toBeNull())
    expect($('[data-testid="integration-disable"]')!.hasAttribute('disabled')).toBe(true)
    const input = $('[data-testid="integration-reason"] textarea') as HTMLTextAreaElement
    input.value = 'Плановые работы 1С'
    input.dispatchEvent(new Event('input'))
    await until(() => expect($('[data-testid="integration-disable"]')!.hasAttribute('disabled')).toBe(false))
    $('[data-testid="integration-disable"]')!.click()
    await until(() => expect($('[data-testid="integration-done"]')).not.toBeNull())
    const post = calls.find((c) => c.method === 'POST' && c.path === '/api/v1/ops/integrations/onec/state')
    expect(post?.body).toMatchObject({ state: 'disabled', reason: { text: 'Плановые работы 1С' }, policy_seq: 7, workplace_id: 'WP-ADM' })
    w.unmount()
  })

  it('отказ гарда — текст ошибки в окне', async () => {
    serve(409)
    const w = await mountAt('/desk?open=integration:visionqc')
    await until(() => expect($('[data-testid="integration-enable"]')).not.toBeNull())
    const input = $('[data-testid="integration-reason"] textarea') as HTMLTextAreaElement
    input.value = 'x'
    input.dispatchEvent(new Event('input'))
    await until(() => expect($('[data-testid="integration-enable"]')!.hasAttribute('disabled')).toBe(false))
    $('[data-testid="integration-enable"]')!.click()
    await until(() => expect($('[data-testid="integration-error"]')).not.toBeNull())
    w.unmount()
  })

  it('неустановленная — пояснение, кнопок нет', async () => {
    serve()
    const w = await mountAt('/desk?open=integration:galaktika')
    await until(() => expect($('[data-testid="integration-not-installed"]')).not.toBeNull())
    expect($('[data-testid="integration-check"]')).toBeNull()
    w.unmount()
  })
})
