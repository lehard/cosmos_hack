// Путь контролёра на столе от сигнала до подписанного решения (эпик 11, «Что
// ожидаем в итоге»): очередь → карточка → панель решений → окно подписи
// уровня 2 → команда через сгенерированный клиент → квитанция с номером
// критического действия. Сервер подменён ответами в форме контракта (режим
// fixtures); интерфейс ходит только через API (FR-150).
import { defineComponent, h, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { provideSigningPort, type SignRequest } from '@/features/sign-decision'
import { i18n } from '@/shared/i18n'
import { flangePassport } from '@/entities/item/__tests__/fixtures'
import { ncCard, queueRows } from '@/entities/nonconformity/__tests__/fixtures'
import DecisionQueueWidget from '@/widgets/decision-queue/ui/DecisionQueueWidget.vue'
import ItemPassportWidget from '@/widgets/item-passport/ui/ItemPassportWidget.vue'
import NcCardWidget from '@/widgets/nc-card/ui/NcCardWidget.vue'
import DecisionPanelWidget from '../ui/DecisionPanelWidget.vue'

const session = {
  demo: true,
  policy_seq: 40,
  role: { id: 'quality_inspector', title: 'Контролёр качества' },
  user: { id: 'qc-03', name: 'Контролёр 03' },
  workplace: { id: 'WP-QC-1', title: 'Пост ОТК 1' },
}
const permissions = (subject: string) => ({
  items: [
    { action: 'nonconformity.nonconformity.confirm', action_class: 'protective', subject },
    { action: 'nonconformity.signal.reject', action_class: 'permissive', subject },
    { action: 'nonconformity.recheck.request', action_class: 'protective', subject },
    { action: 'nonconformity.item.isolate', action_class: 'protective', subject },
  ],
  policy_seq: 40,
})
const receipt = { command_id: 'x', event_ids: ['d-2'], replayed: false, seq: 1270, ca_ref: 'CA-312' }

const json = (body: unknown) => new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json', 'Ant-Backend': 'fixtures' } })
const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
  const u = String(url)
  if (init?.method === 'POST') return json({ ...receipt, command_id: JSON.parse(String(init.body)).command_id })
  if (u.startsWith('/api/v1/auth/session')) return json(session)
  if (u.startsWith('/api/v1/decision-queue')) return json({ items: queueRows() })
  if (u.startsWith('/api/v1/nonconformities/NC-0142')) return json(ncCard())
  if (u.startsWith('/api/v1/items/ENT:FL-0042/passport')) return json(flangePassport())
  if (u.startsWith('/api/v1/permissions')) return json(permissions(u.includes('subject=item') ? 'item' : 'nonconformity'))
  return new Response(JSON.stringify({ code: 'api.not_found', title: 'нет', status: 404 }), { status: 404 })
})

const Desk = defineComponent({
  setup: () => () =>
    h('div', [
      h(DecisionQueueWidget, { widgetId: 'decision-queue', titleKey: 'desks.decisionQueue', slotId: 'queue', slice: { sort: ['risk', 'deadline'] }, density: 'comfortable' }),
      h(NcCardWidget, { widgetId: 'nc-card', titleKey: 'desks.ncCard', slotId: 'card', slice: { view: 'evidence' }, density: 'comfortable' }),
      h(DecisionPanelWidget, { widgetId: 'decision-panel', titleKey: 'decisions.panelTitle', slotId: 'decision', slice: {}, density: 'comfortable' }),
      h(ItemPassportWidget, { widgetId: 'item-passport', titleKey: 'desks.passport', slotId: 'passport', slice: { view: 'compact' }, density: 'comfortable' }),
    ]),
})

beforeEach(() => {
  setActivePinia(createPinia())
  vi.stubGlobal('fetch', fetchMock)
  fetchMock.mockClear()
})
afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

async function mountDesk(root: ReturnType<typeof defineComponent> = Desk) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/items/:id', name: 'item', component: { template: '<div />' } },
      { path: '/nonconformities/:id', name: 'nonconformity', component: { template: '<div />' } },
    ],
  })
  await router.push('/')
  return mount(root, { attachTo: document.body, global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] } })
}

/** Выбрать действие, написать причину и открыть окно подписи. */
async function decide(w: Awaited<ReturnType<typeof mountDesk>>, action: string, reason: string): Promise<HTMLElement> {
  await vi.waitFor(() => expect(w.find(`[data-action="${action}"]`).exists()).toBe(true), { timeout: 10_000 })
  await w.find(`[data-action="${action}"]`).trigger('click')
  await w.find('[data-testid="reason"] textarea').setValue(reason)
  await vi.waitFor(() => expect(w.find('[data-testid="sign"]').attributes('disabled')).toBeUndefined(), { timeout: 10_000 })
  await w.find('[data-testid="sign"]').trigger('click')
  await flushPromises()
  return document.querySelector('[data-testid="sign-confirm"]') as HTMLElement
}

describe('путь контролёра от сигнала до подписанного решения', () => {
  it('очередь → карточка → отклонить сигнал с причиной → окно уровня 2 → квитанция', async () => {
    const w = await mountDesk()

    // Очередь пришла, первая строка выбрана: карточка, панель и паспорт — того же изделия.
    await vi.waitFor(() => expect(w.find('[data-testid="nc-card"]').exists()).toBe(true), { timeout: 10_000 })
    expect(w.find('[data-widget="nc-card"]').attributes('data-mode')).toBe('fixtures')
    await vi.waitFor(() => expect(w.find('[data-testid="item-passport"]').exists()).toBe(true), { timeout: 10_000 })
    expect(w.find('[data-testid="passport-head"]').text()).toContain('FL-0042')

    // Отклонить сигнал: без причины подписать нельзя, с причиной — окно подписи.
    await vi.waitFor(() => expect(w.find('[data-action="reject_signal"]').exists()).toBe(true), { timeout: 10_000 })
    await w.find('[data-action="reject_signal"]').trigger('click')
    await vi.waitFor(() => expect(w.find('[data-testid="sign"]').exists()).toBe(true), { timeout: 10_000 })
    expect(w.find('[data-testid="sign"]').attributes('disabled')).toBeDefined()
    await w.find('[data-testid="reason"] textarea').setValue('Блик на кромке, на повторном снимке признаков нет')
    await vi.waitFor(() => expect(w.find('[data-testid="sign"]').attributes('disabled')).toBeUndefined(), { timeout: 10_000 })
    await w.find('[data-testid="sign"]').trigger('click')
    await flushPromises()

    // Окно уровня 2: сводка решения; агента нет, демо-профиль — подтверждение без агента.
    const dialog = document.querySelector('[data-testid="sign-confirm"]') as HTMLElement
    expect(dialog).not.toBeNull()
    expect(dialog.textContent).toContain('Отклонить сигнал — изделие продолжает маршрут')
    expect(dialog.textContent).toContain('НС-0142')
    ;(dialog.querySelector('[data-testid="confirm-unsigned"]') as HTMLButtonElement).click()
    await vi.waitFor(() => expect(w.find('[data-testid="receipt"]').exists()).toBe(true), { timeout: 10_000 })

    // Команда ушла сгенерированным клиентом с полями контракта.
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')!
    expect(post[0]).toBe('/api/v1/items/ENT:FL-0042/signals/reject')
    const body = JSON.parse(String(post[1]!.body))
    expect(body).toMatchObject({ basis_seq: 1260, policy_seq: 40, workplace_id: 'WP-QC-1', signal_ids: ['SIG-77'], reason: { text: 'Блик на кромке, на повторном снимке признаков нет' } })
    expect(body.command_id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7/)
    expect(body.signature).toBeUndefined()
    expect(w.find('[data-testid="receipt"]').text()).toBe('Решение записано в журнал: запись № 1270 · CA-312')
    w.unmount()
  })

  it('агент токена подписал уровнем 2 — команда уходит с конвертом DSSE (AD-13, AD-14)', async () => {
    const requests: SignRequest[] = []
    const envelope = { payload: 'e30=', payloadType: 'application/vnd.ant.event+json; v=1', signatures: [{ keyid: 'qc-03@1', sig: 'c2ln' }] }
    const WithAgent = defineComponent({
      setup() {
        provideSigningPort({
          status: ref('inserted' as const),
          sign: async (r) => {
            requests.push(r)
            return envelope
          },
        })
        return () => h(Desk)
      },
    })
    const w = await mountDesk(WithAgent)
    const dialog = await decide(w, 'confirm_nc', 'Пора в шве подтверждена повторным снимком')
    expect(dialog.querySelector('[data-testid="confirm-unsigned"]')).toBeNull()
    ;(dialog.querySelector('[data-testid="confirm-token"]') as HTMLButtonElement).click()
    await vi.waitFor(() => expect(w.find('[data-testid="receipt"]').exists()).toBe(true), { timeout: 10_000 })
    expect(requests[0]).toMatchObject({ level: 2, payload_type: 'application/vnd.ant.event+json; v=1', event_type: 'decision.nonconformity.confirmed' })
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')!
    expect(post[0]).toBe('/api/v1/nonconformities/NC-0142/confirm')
    expect(JSON.parse(String(post[1]!.body))).toMatchObject({ severity: 'major', signal_ids: ['SIG-77'], signature: envelope })
    w.unmount()
  })
})
