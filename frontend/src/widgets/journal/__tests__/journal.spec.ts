// Общий журнал (PRD §3a; AD-2, AD-44, AD-46; FR-68, FR-140; кейс §7.2): полоса
// целостности, виды записей различимы, источник и подпись словами, отбор уходит
// в запрос, срез entry_kind, «Показать ещё» по курсору, окно записи.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mountWidget, settle } from '@/entities/run/__tests__/api'
import type { JournalEntryView } from '@/shared/api/generated/model'
import JournalWidget from '../ui/JournalWidget.vue'

const drillOpen = vi.hoisted(() => vi.fn(() => true))
vi.mock('@/features/drill-down', () => ({ useDrillDown: () => ({ open: drillOpen, canOpen: () => true, openPage: () => true }) }))
import { dataRows, eventTypeGroups, shortDigest, sourceText } from '../model/labels'

afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

const props = { widgetId: 'journal', titleKey: 'desks.journal', density: 'compact' }
const REPORT = 'streebog256:0123456789abcdef0123456789abcdef'

const entry = (over: Partial<JournalEntryView> = {}): JournalEntryView => ({
  seq: 101,
  event_id: 'ev-101',
  event_type: 'quality.defect.identified',
  schema_version: 1,
  entry_kind: 'fact',
  chain: 'main',
  stream: 'item:ENT01:F-031',
  item_id: 'ENT01:F-031',
  occurred_at: '2026-09-26T08:59:00Z',
  received_at: '2026-09-26T09:00:00Z',
  recorded_at: '2026-09-26T09:00:00Z',
  committed_at: '2026-09-26T09:00:01Z',
  provenance_class: 'device',
  source_id: 'CAM-3',
  source_kind: 'camera',
  signature_status: 'valid',
  signers: ['cam-3@1'],
  causation_id: null,
  correlation_id: 'ev-101',
  data: { summary: 'Пора на шве', zone: { x: 1, y: 2 } },
  ...over,
})

const page1 = [
  entry(),
  entry({ seq: 102, event_id: 'ev-102', event_type: 'decision.concession.granted', entry_kind: 'decision', source_kind: undefined, source_id: 'QC-01', provenance_class: 'personal', signature_status: 'unverifiable', signers: ['qc-01@2'], ca_ref: 'CA-7', causation_id: 'ev-101', basis_seq: 101, occurred_at: '2026-09-26T09:00:00Z' }),
  entry({ seq: 103, event_id: 'ev-103', event_type: 'journal.genesis.recorded', entry_kind: 'service', item_id: undefined, source_kind: undefined, signature_status: 'not_checked', signers: [], data: undefined, occurred_at: '2026-09-26T09:00:00Z' }),
]
const page2 = [entry({ seq: 104, event_id: 'ev-104', signature_status: 'invalid', occurred_at: '2026-09-26T09:00:00Z' })]

interface Call {
  path: string
  query: URLSearchParams
}

/** Подмена сети: журнал отвечает страницами по курсору, остальное — заготовки. */
function stubApi(opts: { integrity?: unknown; entries?: (q: URLSearchParams) => unknown } = {}): Call[] {
  const calls: Call[] = []
  const now = new Date().toISOString()
  vi.stubGlobal('fetch', async (url: string) => {
    const u = new URL(url, 'http://ant.local')
    calls.push({ path: u.pathname, query: u.searchParams })
    const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json', 'Ant-Backend': 'fixtures' } })
    switch (u.pathname) {
      case '/api/v1/integrity':
        return json(opts.integrity ?? { status: 'ok', server_side: true, interval_seconds: 300, checked_at: now, report_ref: REPORT })
      case '/api/v1/journal/head':
        return json({ seq: 104, ca_seq: 7, clock_mode: 'system', recorded_at: now })
      case '/api/v1/verifier-reports':
        return json({ items: [{ report_digest: REPORT, checked_at: now, checked_up_to_seq: 100, server_side: true, verdict: 'intact' }] })
      case '/api/v1/journal':
        if (opts.entries) return json(opts.entries(u.searchParams))
        return json(u.searchParams.get('cursor') === '103' ? { items: page2 } : { items: page1, next_cursor: '103' })
      default:
        return json({ type: 'urn:ant:problem:api.not_implemented', title: 'x', status: 501, code: 'api.not_implemented' }, 501)
    }
  })
  return calls
}

const journalCalls = (calls: Call[]) => calls.filter((c) => c.path === '/api/v1/journal')

describe('журнал — полоса целостности', () => {
  it('вердикт словами, когда проверено и до какой записи, отпечаток сокращён (полный — в подсказке), голова журнала', async () => {
    stubApi()
    const w = await mountWidget(JournalWidget, props)
    const bar = w.find('[data-testid="integrity-bar"]')
    expect(bar.attributes('data-status')).toBe('ok')
    expect(bar.find('[data-testid="verdict"]').text()).toBe('Цепочка записей проверена — цело')
    expect(bar.find('[data-testid="verdict"]').attributes('data-tone')).toBe('success')
    expect(bar.find('[data-testid="checked-at"]').text()).toContain('проверено')
    expect(bar.find('[data-testid="checked-at"]').text()).toContain('до записи № 100')
    const ref = bar.find('[data-testid="report-ref"]')
    expect(ref.text()).toContain('streebog256:0123456789ab…')
    expect(ref.attributes('title')).toContain(REPORT)
    expect(bar.find('[data-testid="head-seq"]').text()).toBe('Записей: 104')
    expect(bar.find('[data-testid="head-ca"]').text()).toBe('Критических действий: CA-7')
    expect(bar.text()).toContain('независимый верификатор')
  })

  it('нарушение — критический тон и признак на рамке', async () => {
    stubApi({ integrity: { status: 'violated', server_side: true, interval_seconds: 300, checked_at: new Date().toISOString() } })
    const w = await mountWidget(JournalWidget, props)
    expect(w.find('[data-testid="verdict"]').attributes('data-tone')).toBe('critical')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('defect_indication')
  })

  it('«цело», но отчёт старше двух интервалов — «нет свежей проверки», не «цело»', async () => {
    stubApi({ integrity: { status: 'ok', server_side: true, interval_seconds: 60, checked_at: '2026-01-01T00:00:00Z' } })
    const w = await mountWidget(JournalWidget, props)
    expect(w.find('[data-testid="integrity-bar"]').attributes('data-status')).toBe('stale')
    expect(w.find('[data-testid="verdict"]').text()).toContain('Нет свежей проверки')
  })
})

describe('журнал — записи', () => {
  it('вид записи, событие словами и код, источник, подпись словами с подписантами, CA', async () => {
    stubApi()
    const w = await mountWidget(JournalWidget, props)
    const fact = w.find('tr[data-seq="101"]')
    expect(fact.text()).toContain('Входящий факт')
    expect(fact.text()).toContain('quality.defect.identified')
    expect(fact.text()).toContain('Камера')
    expect(fact.text()).toContain('подпись действительна')
    expect(fact.text()).toContain('cam-3@1')
    expect(fact.text()).toContain('F-031')
    expect(fact.text()).toContain('произошло')
    const decision = w.find('tr[data-seq="102"]')
    expect(decision.text()).toContain('Решение человека')
    expect(decision.text()).toContain('Разрешение на отклонение выдано')
    expect(decision.text()).toContain('проверить нельзя')
    expect(decision.text()).toContain('CA-7')
    expect(decision.text()).not.toContain('произошло')
    const service = w.find('tr[data-seq="103"]')
    expect(service.text()).toContain('Служебная запись')
    expect(service.text()).toContain('не проверялась')
    expect(w.find('[data-testid="total"]').text()).toContain('3')
  })

  it('«Показать ещё» дочитывает страницу по курсору; последняя — «показаны все»', async () => {
    const calls = stubApi()
    const w = await mountWidget(JournalWidget, props)
    await w.find('[data-action="more"]').trigger('click')
    await settle()
    expect(journalCalls(calls).at(-1)!.query.get('cursor')).toBe('103')
    expect(w.find('tr[data-seq="104"]').text()).toContain('подпись недействительна')
    expect(w.findAll('tbody tr')).toHaveLength(4)
    expect(w.find('[data-action="more"]').exists()).toBe(false)
    expect(w.text()).toContain('Показаны все записи')
  })

  it('срез entry_kind: decision — переключатель на решениях, вид уходит в запрос', async () => {
    const calls = stubApi({ entries: () => ({ items: [] }) })
    const w = await mountWidget(JournalWidget, { ...props, slice: { entry_kind: 'decision' } })
    expect(journalCalls(calls)[0]!.query.get('entry_kind')).toBe('decision')
    expect(w.find('[data-kind="decision"] input').element).toHaveProperty('checked', true)
    expect(w.text()).toContain('По этому отбору записей нет')
  })

  it('отбор: вид записи и изделие уходят в запрос, «Все» — без вида', async () => {
    const calls = stubApi()
    const w = await mountWidget(JournalWidget, props)
    expect(journalCalls(calls)[0]!.query.has('entry_kind')).toBe(false)
    await w.find('[data-kind="fact"] input').setValue(true)
    await vi.waitFor(() => expect(journalCalls(calls).at(-1)!.query.get('entry_kind')).toBe('fact'))
    const item = w.find('[data-filter="item"] input')
    await item.setValue('ENT01:F-031')
    await item.trigger('keydown', { key: 'Enter' })
    await vi.waitFor(() => expect(journalCalls(calls).at(-1)!.query.get('item_id')).toBe('ENT01:F-031'))
    await w.find('[data-action="reset"]').trigger('click')
    await vi.waitFor(() => {
      const q = journalCalls(calls).at(-1)!.query
      expect(q.has('entry_kind') || q.has('item_id')).toBe(false)
    })
  })
})

describe('журнал — окно записи', () => {
  it('щелчок по строке — окно «Запись журнала № seq»: времена, подпись с пояснением, цепочка, связи, содержимое', async () => {
    stubApi()
    const w = await mountWidget(JournalWidget, props)
    await w.find('tr[data-seq="102"]').trigger('click')
    await settle()
    const head = document.querySelector('[data-record="journal-entry"] [data-testid="record-drawer-head"]')!
    expect(head.textContent).toContain('Запись журнала')
    expect(head.textContent).toContain('№ 102')
    expect(head.textContent).toContain('Разрешение на отклонение выдано')
    const body = document.querySelector('[data-testid="journal-record"]')!
    expect(body.textContent).toContain('Время возникновения')
    expect(body.textContent).toContain('Время фиксации в цепочке')
    expect(body.textContent).toContain('личная подпись')
    expect(body.textContent).toContain('Это не «подпись действительна»')
    expect(body.textContent).toContain('qc-01@2')
    expect(body.textContent).toContain('Основная цепочка')
    expect(body.textContent).toContain('CA-7')
    // Связь: причина — загруженная запись № 101, основание — запись № 101.
    const toCause = document.querySelector<HTMLElement>('[data-action="open-causation"]')!
    expect(toCause.textContent).toContain('Открыть запись № 101')
    toCause.click()
    await settle()
    expect(document.querySelector('[data-testid="journal-record"]')!.getAttribute('data-seq')).toBe('101')
    expect(document.querySelector('[data-testid="record-summary"]')!.textContent).toContain('Пора на шве')
    const data = document.querySelector('[data-testid="record-data"]')!
    expect(data.textContent).not.toContain('Пора на шве')
    expect(data.querySelector('details pre')!.textContent).toContain('"x": 1')
  })

  it('содержимое без data — «недоступно», не пусто', async () => {
    stubApi()
    const w = await mountWidget(JournalWidget, props)
    await w.find('tr[data-seq="103"]').trigger('click')
    await settle()
    expect(document.querySelector('[data-testid="data-hidden"]')!.textContent).toContain('нет ключа расшифровки или прав')
  })

  it('изделие — ссылка на окно изделия, окно записи не открывается', async () => {
    stubApi()
    const w = await mountWidget(JournalWidget, props)
    await w.find('tr[data-seq="101"] [data-action="open-item"]').trigger('click')
    await settle()
    expect(drillOpen).toHaveBeenCalledWith({ entity: 'item', id: 'ENT01:F-031' })
    expect(document.querySelector('[data-testid="journal-record"]')).toBeNull()
  })
})

describe('журнал — правила показа', () => {
  it('источник: вид источника; вывод системы; решение человека; без вида — «неизвестен»', () => {
    const t = (k: string) => k
    expect(sourceText({ source_kind: 'camera', entry_kind: 'fact' }, t)).toBe('timeline.sourceKind.camera')
    expect(sourceText({ source_kind: 'teleport', entry_kind: 'fact' }, t)).toBe('UNKNOWN(teleport)')
    expect(sourceText({ entry_kind: 'reaction' }, t)).toBe('timeline.sourceKind.system')
    expect(sourceText({ entry_kind: 'decision' }, t)).toBe('timeline.layer.decision')
    expect(sourceText({ entry_kind: 'fact' }, t)).toBe('widgets.passport.source.unknown')
  })

  it('отпечаток сокращается с алгоритмом; содержимое — простое и JSON', () => {
    expect(shortDigest('streebog256:abcdef0123456789ff')).toBe('streebog256:abcdef012345…')
    expect(shortDigest('short')).toBe('short')
    expect(dataRows({ a: 1, b: null, c: [1] })).toEqual([
      { key: 'a', text: '1', json: null },
      { key: 'b', text: '—', json: null },
      { key: 'c', text: null, json: '[\n  1\n]' },
    ])
  })

  it('типы событий — по семействам со словарём, первым — всё семейство', () => {
    const groups = eventTypeGroups((k, named) => (named ? `${k}:${JSON.stringify(named)}` : k), () => true)
    const quality = groups.find((g) => g.key === 'quality')!
    expect(quality.children[0]!.value).toBe('quality.')
    expect(quality.children.some((c) => c.value === 'quality.defect.identified')).toBe(true)
  })
})
