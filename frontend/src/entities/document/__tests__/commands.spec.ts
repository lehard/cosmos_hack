// Команды над документом (эпик 28): отказ с замечанием, печать с QR,
// заверение бумажной подписи, «Запросить решение» — через сгенерированный
// клиент, тела по схемам contracts/openapi.yaml, ответ в конверте {data, headers}.
// Сервер подменён ответами в форме контракта (режим fixtures).
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { documentKeys, useAttestPaper, useDeclineDocument, usePrintPaper, useRequestDecision } from '@/entities/document'

const DIGEST = 'streebog256:c0ffee'
const receipt = { command_id: 'c-1', event_ids: ['ev-1'], replayed: false, seq: 1301 }
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json', 'Ant-Backend': 'fixtures' } })

const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
  const u = String(url)
  if (init?.method === 'POST' && u.startsWith('/api/v1/materials')) {
    return json({ material_address: 'streebog256:5ca9', kind: 'scan', media_type: 'image/png', size_bytes: 3, is_illustration: false }, 201)
  }
  if (init?.method === 'POST' && u.endsWith('/print')) {
    return json({ ...receipt, version: 1, doc_digest: DIGEST, qr: `ant:doc:DOC-NCD-142:${DIGEST}`, print_url: '/api/v1/documents/DOC-NCD-142/print?version=1' })
  }
  if (init?.method === 'POST' && u === '/api/v1/documents/versions') return json({ ...receipt, document_id: 'DOC-REQ-7', version: 1 })
  if (init?.method === 'POST') return json(receipt)
  if (u.startsWith('/api/v1/documents/DOC-NCD-142/print')) {
    return json({ document_id: 'DOC-NCD-142', version: 1, doc_digest: DIGEST, html: '<html>печатная форма</html>', qr: 'q', qr_svg: '<svg/>', rendering_hash: 'h', printed_at: '2026-09-26T10:00:00Z' })
  }
  return json({ code: 'api.not_found', title: 'нет', status: 404 }, 404)
})

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
  fetchMock.mockClear()
})
afterEach(() => vi.unstubAllGlobals())

/** Запустить композиции команд внутри приложения с Vue Query. */
function setup() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries')
  let api!: {
    decline: ReturnType<typeof useDeclineDocument>
    print: ReturnType<typeof usePrintPaper>
    attest: ReturnType<typeof useAttestPaper>
    request: ReturnType<typeof useRequestDecision>
  }
  mount(
    defineComponent({
      setup() {
        api = { decline: useDeclineDocument(), print: usePrintPaper(), attest: useAttestPaper(), request: useRequestDecision() }
        return () => null
      },
    }),
    { global: { plugins: [[VueQueryPlugin, { queryClient }]] } },
  )
  return { api, invalidate }
}

const call = (i: number) => {
  const [url, init] = fetchMock.mock.calls[i] as [string, RequestInit | undefined]
  return { url: String(url), method: init?.method ?? 'GET', init, body: typeof init?.body === 'string' ? JSON.parse(init.body) : init?.body }
}
const header = { basis_seq: 169999, policy_seq: 224, workplace_id: 'WP-QC-1' }

describe('команды над документом через сгенерированный клиент', () => {
  it('«Не согласовать»: замечание, этап, отпечаток и заголовок команды; квитанция в конверте', async () => {
    const { api, invalidate } = setup()
    const res = await api.decline.mutateAsync({ document_id: 'DOC-NCD-142', version: 1, stage: 2, comment: 'Нужен протокол', doc_digest: DIGEST, ...header })
    const c = call(0)
    expect(c.method).toBe('POST')
    expect(c.url).toBe('/api/v1/documents/DOC-NCD-142/route/declines')
    expect(c.body).toMatchObject({ version: 1, stage: 2, comment: 'Нужен протокол', doc_digest: DIGEST, ...header })
    expect(c.body.command_id).toMatch(/^[0-9a-f-]{36}$/)
    expect(res.data.seq).toBe(1301)
    expect(res.headers?.get('Ant-Backend')).toBe('fixtures')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: documentKeys.one('DOC-NCD-142') })
  })

  it('печать с QR: запись «напечатан», затем печатная форма той версии, что вернул сервер', async () => {
    const { api } = setup()
    const res = await api.print.mutateAsync({ document_id: 'DOC-NCD-142', version: 1, ...header, command_id: 'c-print' })
    expect(call(0)).toMatchObject({ method: 'POST', url: '/api/v1/documents/DOC-NCD-142/print', body: { version: 1, command_id: 'c-print', ...header } })
    expect(call(1)).toMatchObject({ method: 'GET', url: '/api/v1/documents/DOC-NCD-142/print?version=1' })
    expect(res.data.print_url).toBe('/api/v1/documents/DOC-NCD-142/print?version=1')
    expect(res.data.html).toBe('<html>печатная форма</html>')
  })

  it('заверение: скан — в хранилище материалов (вид scan), затем заверение с адресом скана и учётным номером', async () => {
    const { api } = setup()
    const file = new File([new Uint8Array([1, 2, 3])], 'scan.png', { type: 'image/png' })
    await api.attest.mutateAsync({
      document_id: 'DOC-NCD-142',
      version: 1,
      stage: 2,
      file,
      archive_no: 'ОТК-2026-0142',
      signer_person_id: 'vp-01',
      doc_digest: DIGEST,
      item_id: 'ENT:FL-0042',
      ...header,
    })
    const up = call(0)
    expect(up.method).toBe('POST')
    expect(up.url).toBe('/api/v1/materials?kind=scan&item_id=ENT%3AFL-0042')
    expect(new Headers(up.init?.headers).get('Content-Type')).toBe('image/png')
    expect(up.body).toBe(file)
    const at = call(1)
    expect(at.url).toBe('/api/v1/documents/DOC-NCD-142/paper-signatures')
    expect(at.body).toMatchObject({
      version: 1,
      stage: 2,
      paper_original_no: 'ОТК-2026-0142',
      scan_address: 'streebog256:5ca9',
      signer_person_id: 'vp-01',
      doc_digest: DIGEST,
      ...header,
    })
    expect(at.body).not.toHaveProperty('file')
    expect(at.body).not.toHaveProperty('archive_no')
  })

  it('«Запросить решение»: действие и объект, шаблон выбирает сервер', async () => {
    const { api, invalidate } = setup()
    const res = await api.request.mutateAsync({ action: 'nonconformity.disposition.record', subject_ref: 'nonconformity:NC-0142', ...header })
    expect(call(0)).toMatchObject({
      method: 'POST',
      url: '/api/v1/documents/versions',
      body: { action: 'nonconformity.disposition.record', subject_ref: 'nonconformity:NC-0142', ...header },
    })
    expect(res.data.document_id).toBe('DOC-REQ-7')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: documentKeys.list() })
  })

  it('ошибка сервера доходит до виджета как problem+json', async () => {
    fetchMock.mockResolvedValueOnce(json({ code: 'access.forbidden', title: 'Нет полномочий', status: 403 }, 403))
    const { api } = setup()
    await expect(api.decline.mutateAsync({ document_id: 'DOC-NCD-142', version: 1, stage: 2, comment: 'x' })).rejects.toMatchObject({
      status: 403,
      info: { code: 'access.forbidden' },
    })
  })
})
