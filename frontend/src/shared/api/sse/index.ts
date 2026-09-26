/**
 * Живые обновления столов (AD-21) — единственный ручной сетевой модуль интерфейса
 * (исключение ESLint из AD-20). Канал `sse` из contracts/events/asyncapi.yaml:
 * `id:` = seq, `event: entity_changed`, `data:` — EntityChanged (тип — из
 * сгенерированного клиента). Данных сущности в сообщении нет: инвалидируем ключи
 * Vue Query `[сущность, id]` и `[сущность, LIST]` — виджеты перечитают через API.
 * Переподключение и Last-Event-ID делает сам EventSource.
 */
import type { QueryClient } from '@tanstack/vue-query'
import { readonly, ref, type Ref } from 'vue'
import { EntityKind, type EntityChanged } from '../generated/model'
import { LIST, PERMISSIONS } from '../keys'
import { SSE_ADDRESS } from '../generated/stream'

/** Имя события SSE с изменением сущности. */
export const ENTITY_CHANGED_EVENT = 'entity_changed'

/** Состояние канала — для шапки: прерванный канал значит «данные могут отставать». */
export type LiveStatus = 'connecting' | 'open' | 'closed'

/** Запущенный канал живых обновлений. */
export interface LiveUpdates {
  status: Readonly<Ref<LiveStatus>>
  /** seq последнего применённого сообщения. */
  lastSeq: Readonly<Ref<number | null>>
  stop: () => void
}

const KINDS = new Set<string>(Object.values(EntityKind))

/** Разбор `data:` сообщения; чужое или битое — null (сообщение пропускается). */
export function parseEntityChanged(raw: string): EntityChanged | null {
  let msg: unknown
  try {
    msg = JSON.parse(raw)
  } catch {
    return null
  }
  if (!msg || typeof msg !== 'object') return null
  const m = msg as Partial<EntityChanged>
  if (typeof m.entity !== 'string' || !KINDS.has(m.entity)) return null
  if (typeof m.id !== 'string' || typeof m.seq !== 'number') return null
  return m as EntityChanged
}

/**
 * Инвалидация кэша по сообщению (соглашение о ключах — shared/api/keys.ts).
 * Изменение политики дополнительно сбрасывает все запросы прав (AD-15: список
 * действий на экране совпадает с тем, что разрешит сервер).
 */
export async function applyEntityChanged(queryClient: QueryClient, msg: EntityChanged): Promise<void> {
  const jobs = [
    queryClient.invalidateQueries({ queryKey: [msg.entity, msg.id] }),
    queryClient.invalidateQueries({ queryKey: [msg.entity, LIST] }),
  ]
  if (msg.entity === 'policy') {
    jobs.push(queryClient.invalidateQueries({ predicate: (q) => q.queryKey.includes(PERMISSIONS) }))
  }
  await Promise.all(jobs)
}

/**
 * Открыть канал живых обновлений.
 * @param queryClient — кэш, в котором инвалидируются ключи
 * @param url — адрес канала (по умолчанию — из contracts/events/asyncapi.yaml)
 */
export function startLiveUpdates(queryClient: QueryClient, url: string = SSE_ADDRESS): LiveUpdates {
  const status = ref<LiveStatus>('connecting')
  const lastSeq = ref<number | null>(null)
  const source = new EventSource(url, { withCredentials: true })
  source.onopen = () => {
    status.value = 'open'
  }
  source.onerror = () => {
    // EventSource переподключается сам; CLOSED — окончательно (например, 401).
    status.value = source.readyState === EventSource.CLOSED ? 'closed' : 'connecting'
  }
  source.addEventListener(ENTITY_CHANGED_EVENT, (event) => {
    const msg = parseEntityChanged((event as MessageEvent<string>).data)
    if (!msg) return
    lastSeq.value = msg.seq
    void applyEntityChanged(queryClient, msg)
  })
  return {
    status: readonly(status),
    lastSeq: readonly(lastSeq),
    stop: () => {
      source.close()
      status.value = 'closed'
    },
  }
}
