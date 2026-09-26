// Мост к расширению «Главный — подпись» (AD-14, Д-72): состояние ключа для
// шапки и подпись через порт; без расширения — «агент не найден».
import { afterEach, describe, expect, it } from 'vitest'
import { createExtensionSigningPort, payloadTypeOf } from '@/features/sign-decision'
import { extensionRequest, refreshTokenStatus, setTokenPerson, useTokenInfo, useTokenStatus } from '@/shared/lib/token-agent'

type Req = { type: string; request_id: string; sign?: { level: number; event_type?: string } }
type Person = { id: string; name?: string } | null | undefined
let handler: ((ev: MessageEvent) => void) | null = null

/** Поддельный content-скрипт расширения: отвечает на запросы страницы. */
function fakeExtension(reply: (rq: Req, person: Person) => unknown): void {
  handler = (ev: MessageEvent) => {
    const d = ev.data as { source?: string; id?: number; request?: Req; person?: Person }
    if (d?.source !== 'glavny-page' || !d.request) return
    const response = reply(d.request, d.person)
    window.dispatchEvent(new MessageEvent('message', { data: { source: 'glavny-ext', id: d.id, response }, source: window, origin: window.location.origin }))
  }
  window.addEventListener('message', handler)
}

afterEach(() => {
  if (handler) window.removeEventListener('message', handler)
  handler = null
  setTokenPerson(null)
})

describe('мост к расширению', () => {
  it('ключ загружен и разблокирован — «готов», в подсказке владелец и класс хранения', async () => {
    fakeExtension((rq) => ({
      protocol_version: 1,
      request_id: rq.request_id,
      type: 'status',
      status: { token_present: true, pin_unlocked: true, person_id: 'INS-01', key_storage: 'software_browser', storage_variant: 'extension' },
    }))
    expect(await refreshTokenStatus()).toBe('inserted')
    expect(useTokenStatus().value).toBe('inserted')
    expect(useTokenInfo().value?.key_storage).toBe('software_browser')
  })

  it('ключ загружен, PIN не введён — «заблокирован»; ключа нет — «не загружен»', async () => {
    let unlocked = false
    let present = true
    fakeExtension((rq) => ({ protocol_version: 1, request_id: rq.request_id, type: 'status', status: { token_present: present, pin_unlocked: unlocked } }))
    expect(await refreshTokenStatus()).toBe('locked')
    present = false
    expect(await refreshTokenStatus()).toBe('missing')
    unlocked = true
  })

  it('подпись: конверт из ответа расширения; отказ уровня 1 — ошибка с кодом агента', async () => {
    const envelope = { payloadType: payloadTypeOf('event'), payload: 'e30=', signatures: [{ keyid: 'ins-01-ta@1', sig: 'AA==' }] }
    fakeExtension((rq) =>
      rq.sign?.level === 1
        ? { protocol_version: 1, request_id: rq.request_id, type: 'error', error: { code: 'signing.level_not_allowed', message: 'нужна подпись уровня 2' } }
        : { protocol_version: 1, request_id: rq.request_id, type: 'signed', signed: [{ envelope, local_journal_seq: 1, client_signed_at: '2026-09-26T10:00:00.000Z' }] },
    )
    const port = createExtensionSigningPort()
    await expect(port.sign({ level: 2, payload_type: payloadTypeOf('event'), payload_b64: 'e30=', event_type: 'decision.nonconformity.confirmed' })).resolves.toEqual(envelope)
    await expect(port.sign({ level: 1, payload_type: payloadTypeOf('event'), payload_b64: 'e30=', event_type: 'inspection.result.recorded' })).rejects.toMatchObject({
      info: { code: 'signing.level_not_allowed' },
    })
  })

  it('ключи многих персон: страница сообщает, кто вошёл, состояние — по его ключу', async () => {
    const loaded = ['INS-01', 'HQC-01']
    fakeExtension((rq, person) => ({
      protocol_version: 1,
      request_id: rq.request_id,
      type: 'status',
      status: { token_present: !!person && loaded.includes(person.id), pin_unlocked: true, person_id: person?.id, persons: loaded },
    }))
    setTokenPerson({ id: 'INS-01', name: 'Контролёр ОТК' })
    expect(await refreshTokenStatus()).toBe('inserted')
    expect(useTokenInfo().value?.person_id).toBe('INS-01')
    setTokenPerson({ id: 'PM-01' })
    expect(await refreshTokenStatus()).toBe('missing')
    expect(useTokenInfo().value?.person_id).toBe('PM-01')
  })

  it('расширения нет — «агент не найден» по таймауту', async () => {
    const r = await extensionRequest({ type: 'status' }, 50)
    expect(r.error?.code).toBe('signing.agent_not_found')
  })
})
