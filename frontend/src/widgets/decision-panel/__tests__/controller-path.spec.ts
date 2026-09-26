// Путь контролёра на столе от сигнала до подписанного решения (эпик 11, «Что
// ожидаем в итоге»): очередь → щелчок по строке → правое окно несоответствия
// (Д-70: карточка, внизу — панель решений) → окно подписи уровня 2 → команда
// через сгенерированный клиент → квитанция с номером критического действия.
// Сервер подменён ответами в форме контракта (режим fixtures); интерфейс ходит
// только через API (FR-150). Окно записи выводится в body — ищем через document.
import { defineComponent, h, provide, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { provideSigningPort, type SignRequest } from '@/features/sign-decision'
import { i18n } from '@/shared/i18n'
import { RECORD_DRAWER } from '@/shared/model/record'
import RecordDrawerHost from '@/app/record/RecordDrawerHost.vue'
import { RECORD_KINDS, recordKinds } from '@/app/record/registry'
import { flangePassport } from '@/entities/item/__tests__/fixtures'
import { ncCard, queueRows } from '@/entities/nonconformity/__tests__/fixtures'
import DecisionQueueWidget from '@/widgets/decision-queue/ui/DecisionQueueWidget.vue'

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

/** Стол контролёра, как его собирает оболочка: очередь и окно записи. */
const Desk = defineComponent({
  setup() {
    provide(RECORD_DRAWER, { kinds: RECORD_KINDS })
    return () =>
      h('div', [
        h(DecisionQueueWidget, { widgetId: 'decision-queue', titleKey: 'desks.decisionQueue', slotId: 'queue', slice: { sort: ['risk', 'deadline'] }, density: 'comfortable' }),
        h(RecordDrawerHost),
      ])
  },
})

const $ = <T extends Element = HTMLElement>(sel: string) => document.querySelector<T>(sel)
const has = (sel: string) => expect($(sel)).not.toBeNull()
// Содержимое окна (виджеты) грузится лениво; под нагрузкой полного прогона — до 10 с.
const until = (check: () => void) => vi.waitFor(check, { timeout: 10_000 })

// Первый импорт содержимого окна (с виджетами) в тестах долгий — грузим заранее.
beforeAll(async () => {
  await Promise.all([
    ...Object.values(recordKinds).map((k) => k.load()),
    import('@/widgets/nc-card'),
    import('@/widgets/decision-panel'),
    import('@/widgets/item-passport'),
  ])
}, 60_000)

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
  const w = mount(root, { attachTo: document.body, global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] } })
  return { w, router }
}

/** Щелчок по первой строке очереди — окно несоответствия с панелью решений внизу. */
async function openFirstRow(ctx: Awaited<ReturnType<typeof mountDesk>>): Promise<void> {
  await until(() => expect(ctx.w.findAll('li.row').length).toBeGreaterThan(0))
  await ctx.w.findAll('li.row')[0]!.trigger('click')
  await flushPromises()
  expect(ctx.router.currentRoute.value.query.open).toBe('nonconformity:NC-0142')
  await until(() => has('[data-record="nonconformity"] [data-testid="record-drawer-actions"]'))
}

/** Написать причину в панели решений окна. */
function typeReason(text: string): void {
  const area = $<HTMLTextAreaElement>('[data-testid="record-drawer-actions"] [data-testid="reason"] textarea')!
  area.value = text
  area.dispatchEvent(new Event('input'))
}

/** Выбрать действие, написать причину и открыть окно подписи. */
async function decide(action: string, reason: string): Promise<HTMLElement> {
  await until(() => has(`[data-action="${action}"]`))
  $(`[data-action="${action}"]`)!.click()
  await flushPromises()
  typeReason(reason)
  await until(() => expect($('[data-testid="sign"]')!.hasAttribute('disabled')).toBe(false))
  $('[data-testid="sign"]')!.click()
  await flushPromises()
  return $('[data-testid="sign-confirm"]') as HTMLElement
}

describe('путь контролёра от сигнала до подписанного решения', () => {
  it('очередь → карточка → отклонить сигнал с причиной → окно уровня 2 → квитанция', async () => {
    const ctx = await mountDesk()

    // Ничего не открыто само; щелчок по строке — окно несоответствия: карточка
    // того же изделия, внизу — панель решений.
    expect($('[data-record]')).toBeNull()
    await openFirstRow(ctx)
    await until(() => has('[data-record="nonconformity"] [data-testid="nc-card"]'))
    expect($('[data-widget="nc-card"]')!.getAttribute('data-mode')).toBe('fixtures')
    expect($('[data-testid="record-drawer-head"]')!.textContent).toContain('НС-0142')

    // Отклонить сигнал: без причины подписать нельзя, с причиной — окно подписи.
    await until(() => has('[data-testid="record-drawer-actions"] [data-action="reject_signal"]'))
    $('[data-action="reject_signal"]')!.click()
    await until(() => has('[data-testid="sign"]'))
    expect($('[data-testid="sign"]')!.hasAttribute('disabled')).toBe(true)
    typeReason('Блик на кромке, на повторном снимке признаков нет')
    await until(() => expect($('[data-testid="sign"]')!.hasAttribute('disabled')).toBe(false))
    $('[data-testid="sign"]')!.click()
    await flushPromises()

    // Окно уровня 2: сводка решения; агента нет, демо-профиль — подтверждение без агента.
    const dialog = $('[data-testid="sign-confirm"]') as HTMLElement
    expect(dialog).not.toBeNull()
    expect(dialog.textContent).toContain('Отклонить сигнал — изделие продолжает маршрут')
    expect(dialog.textContent).toContain('НС-0142')
    ;(dialog.querySelector('[data-testid="confirm-unsigned"]') as HTMLButtonElement).click()
    await until(() => has('[data-testid="receipt"]'))

    // Команда ушла сгенерированным клиентом с полями контракта.
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')!
    expect(post[0]).toBe('/api/v1/items/ENT:FL-0042/signals/reject')
    const body = JSON.parse(String(post[1]!.body))
    expect(body).toMatchObject({ basis_seq: 1260, policy_seq: 40, workplace_id: 'WP-QC-1', signal_ids: ['SIG-77'], reason: { text: 'Блик на кромке, на повторном снимке признаков нет' } })
    expect(body.command_id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7/)
    expect(body.signature).toBeUndefined()
    expect($('[data-testid="receipt-ref"]')!.textContent!.trim()).toBe('Запись журнала № 1270 · CA-312')
    ctx.w.unmount()
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
    const ctx = await mountDesk(WithAgent)
    await openFirstRow(ctx)
    const dialog = await decide('confirm_nc', 'Пора в шве подтверждена повторным снимком')
    expect(dialog.querySelector('[data-testid="confirm-unsigned"]')).toBeNull()
    ;(dialog.querySelector('[data-testid="confirm-token"]') as HTMLButtonElement).click()
    await until(() => has('[data-testid="receipt"]'))
    expect(requests[0]).toMatchObject({ level: 2, payload_type: 'application/vnd.ant.event+json; v=1', event_type: 'decision.nonconformity.confirmed' })
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')!
    expect(post[0]).toBe('/api/v1/nonconformities/NC-0142/confirm')
    expect(JSON.parse(String(post[1]!.body))).toMatchObject({ severity: 'major', signal_ids: ['SIG-77'], signature: envelope })
    ctx.w.unmount()
  })
})
