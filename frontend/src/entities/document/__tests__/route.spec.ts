// Маршрут подписей и бумажная подпись: показ засчитанного, QR, заверитель ≠
// подписант (FR-66, FR-136, FR-139, AD-43).
import { describe, expect, it } from 'vitest'
import { canAttest, paperAllowed, qrPayload, routeProgress } from '@/entities/document'
import { useAsIsRequest } from './fixtures'

describe('маршрут подписей', () => {
  it('засчитанные против нужных; подпись по прежней версии не засчитывается', () => {
    const p = routeProgress(useAsIsRequest().document.route)
    expect(p).toMatchObject({ have: 1, need: 2 })
    expect(p.pending.map((s) => s.stage)).toEqual([2])
    expect(p.previous.map((s) => s.event_id)).toEqual(['s-0'])
  })

  it('бумага разрешена только там, где это задал маршрут', () => {
    const [s1, s2] = useAsIsRequest().document.route
    expect(paperAllowed(s1)).toBe(false)
    expect(paperAllowed(s2)).toBe(true)
    expect(paperAllowed(null)).toBe(false)
  })
})

describe('бумажная подпись (FR-139)', () => {
  it('QR — ant:doc:‹id›:‹отпечаток› по соглашению контракта', () => {
    expect(qrPayload({ document_id: 'DOC-NCD-142', doc_digest: 'streebog256:c0ffee' })).toBe('ant:doc:DOC-NCD-142:streebog256:c0ffee')
  })

  it('заверитель не может быть подписантом', () => {
    expect(canAttest('vp-01', 'vp-01')).toBe(false)
    expect(canAttest('master-07', 'vp-01')).toBe(true)
    expect(canAttest(null, 'vp-01')).toBe(false)
  })
})
