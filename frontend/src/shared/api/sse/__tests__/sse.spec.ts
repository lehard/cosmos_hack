// Канал живых обновлений (AD-21): сообщение (сущность, id) инвалидирует ключи
// [сущность, id] и [сущность, LIST], не трогая чужие.
import { QueryClient } from '@tanstack/vue-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LIST, PERMISSIONS } from '../../keys'
import { applyEntityChanged, ENTITY_CHANGED_EVENT, parseEntityChanged, startLiveUpdates } from '..'

function client() {
  const qc = new QueryClient()
  for (const key of [
    ['item', 'ENT01:1', { axis: 'occurred' }],
    ['item', LIST, 'lookup', 'q'],
    ['item', 'ENT01:2'],
    ['nonconformity', 'ENT01:1'],
    ['policy', LIST, PERMISSIONS],
    ['item', 'ENT01:2', PERMISSIONS],
  ]) {
    qc.setQueryData(key, { ok: true })
  }
  return qc
}

const invalidated = (qc: QueryClient, key: unknown[]) => qc.getQueryState(key)?.isInvalidated

describe('SSE: разбор сообщения', () => {
  it('принимает сообщение контракта', () => {
    expect(parseEntityChanged('{"entity":"item","id":"ENT01:1","seq":42}')).toEqual({ entity: 'item', id: 'ENT01:1', seq: 42 })
  })
  it('пропускает битое и чужое', () => {
    expect(parseEntityChanged('не json')).toBeNull()
    expect(parseEntityChanged('{"entity":"unknown_kind","id":"1","seq":1}')).toBeNull()
    expect(parseEntityChanged('{"entity":"item","id":1,"seq":1}')).toBeNull()
  })
})

describe('SSE: инвалидация [сущность, id]', () => {
  it('сущность и списки этого вида — да, соседи — нет', async () => {
    const qc = client()
    await applyEntityChanged(qc, { entity: 'item', id: 'ENT01:1', seq: 1 })
    expect(invalidated(qc, ['item', 'ENT01:1', { axis: 'occurred' }])).toBe(true)
    expect(invalidated(qc, ['item', LIST, 'lookup', 'q'])).toBe(true)
    expect(invalidated(qc, ['item', 'ENT01:2'])).toBe(false)
    expect(invalidated(qc, ['nonconformity', 'ENT01:1'])).toBe(false)
  })

  it('изменение политики сбрасывает все запросы прав', async () => {
    const qc = client()
    await applyEntityChanged(qc, { entity: 'policy', id: 'global', seq: 2 })
    expect(invalidated(qc, ['policy', LIST, PERMISSIONS])).toBe(true)
    expect(invalidated(qc, ['item', 'ENT01:2', PERMISSIONS])).toBe(true)
    expect(invalidated(qc, ['item', 'ENT01:2'])).toBe(false)
  })
})

describe('SSE: канал', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('слушает entity_changed на адресе из контракта и закрывается', async () => {
    const listeners: Record<string, (e: MessageEvent<string>) => void> = {}
    const close = vi.fn()
    class FakeEventSource {
      static readonly CLOSED = 2
      readyState = 1
      onopen: (() => void) | null = null
      onerror: (() => void) | null = null
      constructor(public url: string) {
        created.push(url)
      }
      addEventListener(name: string, fn: (e: MessageEvent<string>) => void) {
        listeners[name] = fn
      }
      close = close
    }
    const created: string[] = []
    vi.stubGlobal('EventSource', FakeEventSource)
    const qc = client()
    const live = startLiveUpdates(qc)
    expect(created).toEqual(['/api/v1/stream'])
    listeners[ENTITY_CHANGED_EVENT]!(new MessageEvent(ENTITY_CHANGED_EVENT, { data: '{"entity":"item","id":"ENT01:1","seq":7}' }))
    await vi.waitFor(() => expect(invalidated(qc, ['item', 'ENT01:1', { axis: 'occurred' }])).toBe(true))
    expect(live.lastSeq.value).toBe(7)
    live.stop()
    expect(close).toHaveBeenCalled()
    expect(live.status.value).toBe('closed')
  })
})
