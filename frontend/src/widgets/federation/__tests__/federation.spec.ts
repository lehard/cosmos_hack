// Партнёры и выписки (эпик 41; FR-131, FR-132): статус происхождения входящих,
// квитанция исходящих, приём изменённой выписки — отказ с текстом.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mockApi, mountWidget, problem, settle } from '@/entities/run/__tests__/api'
import FederationWidget from '../ui/FederationWidget.vue'

afterEach(() => vi.unstubAllGlobals())
const props = { widgetId: 'federation', titleKey: 'widgets.federation.title' }
const at = '2026-09-18T05:00:00Z'
const partners = {
  items: [
    { partner_code: 'MZ01', name: 'Металлургический завод МЗ-1', root_fingerprints: ['streebog256:' + 'a'.repeat(64)], document_id: 'DOC-PARTNER-MZ01', registered_at: at, channel: 'ok' },
    { partner_code: 'SB01', name: 'Сборочное предприятие СБ-1', root_fingerprints: ['streebog256:' + 'b'.repeat(64)], document_id: 'DOC-PARTNER-SB01', registered_at: at, channel: 'ok' },
  ],
}
const extracts = {
  items: [
    { extract_digest: 'streebog256:' + '1'.repeat(64), direction: 'incoming', partner_code: 'MZ01', origin_status: 'verified', heat_no: '5512', label: 'Поковки из плавки 5512', subject: { entity: 'lot', id: 'LOT-ZF-201' }, acknowledged: null, at },
    { extract_digest: 'streebog256:' + '2'.repeat(64), direction: 'incoming', partner_code: 'PK02', origin_status: 'unverified', label: 'Кольца', acknowledged: null, at },
    { extract_digest: 'streebog256:' + '3'.repeat(64), direction: 'outgoing', partner_code: 'SB01', origin_status: 'not_applicable', label: 'Фланец Ф-001', acknowledged: true, at },
  ],
}

describe('партнёры и выписки', () => {
  it('входящие — статус происхождения, исходящая — квитанция; непроверенное — не «норма»', async () => {
    mockApi({ 'GET /api/v1/partners': partners, 'GET /api/v1/passport-extracts': extracts })
    const w = await mountWidget(FederationWidget, props)
    const rows = w.findAll('tr[data-digest]')
    expect(rows).toHaveLength(3)
    expect(rows[0]?.text()).toContain('происхождение подтверждено')
    expect(rows[0]?.text()).toContain('5512')
    expect(rows[1]?.attributes('data-origin')).toBe('unverified')
    expect(rows[2]?.text()).toContain('квитанция: получено')
    expect(w.find('tr[data-partner="MZ01"]').text()).toContain('Металлургический завод МЗ-1')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('defect_indication')
  })

  it('изменённая выписка — отказ приёма с текстом', async () => {
    const calls = mockApi({
      'GET /api/v1/partners': partners,
      'GET /api/v1/passport-extracts': extracts,
      'POST /api/v1/passport-extracts/incoming': problem(422, 'federation.extract_tampered'),
    })
    const w = await mountWidget(FederationWidget, props)
    await w.find('[data-testid="federation-receive-open"]').trigger('click')
    await settle()
    // Партнёр подставляется по отправителю из пакета.
    const envelope = JSON.stringify({ payloadType: 'application/vnd.ant.passport-extract+json; v=1', payload: btoa('{"sender":"MZ01"}'), signatures: [] })
    await w.find('[data-testid="receive-envelope"] textarea').setValue(envelope)
    await settle()
    await w.find('[data-testid="receive-submit"]').trigger('click')
    await settle()
    const post = calls.find((c) => c.method === 'POST')
    expect(post?.body).toMatchObject({ partner_code: 'MZ01', envelope })
    expect(w.find('[data-testid="receive-error"]').exists()).toBe(true)
  })
})
