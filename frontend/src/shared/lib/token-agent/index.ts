/**
 * Мост к расширению «Главный — подпись» (AD-14, FR-69, Д-72). Браузер говорит
 * с ключом человека не через сервер, а через расширение: страница шлёт
 * `window.postMessage` в content-скрипт расширения, тот — в фоновую службу;
 * протокол — contracts/internal/token-agent (те же сообщения, что у агента
 * токена по Native Messaging).
 *
 * Состояния для шапки и окна подписи:
 * - `agent_missing` — расширения нет (остаётся подпись на бумаге, AD-43);
 * - `missing` — расширение есть, ключ не загружен (или токен не вставлен);
 * - `locked` — ключ загружен, PIN спросит окно подписи;
 * - `inserted` — ключ загружен и разблокирован: «готов».
 *
 * Расширение держит ключи многих персон (демо из одного браузера, Д-72) и
 * подписывает ключом вошедшего: страница сообщает его в каждом сообщении
 * (`person`, см. `setTokenPerson`), состояние — по его ключу.
 */
import { readonly, ref, type Ref } from 'vue'
import type { TokenAgentRequestV1, TokenAgentResponseV1 } from '@/shared/contracts/procs'

/** Состояние токена на рабочем месте. */
export type TokenStatus = 'inserted' | 'locked' | 'missing' | 'agent_missing'

/** Сведения о ключе из ответа `status` (класс хранения — AD-11). */
export type TokenInfo = NonNullable<TokenAgentResponseV1['status']> & {
  key_storage?: string
  storage_variant?: string
  key_refs?: string[]
  adapter?: 'browser' | 'agent'
  /** Люди, чьи ключи загружены в расширение. */
  persons?: string[]
}

/** Вошедший в «Главный» человек: расширение выбирает его ключ. */
export type TokenPerson = { id: string; name?: string }

const status = ref<TokenStatus>('agent_missing')
const info = ref<TokenInfo | null>(null)

const PAGE = 'glavny-page'
const EXT = 'glavny-ext'
/** Сколько ждать ответа расширения на служебный запрос; подпись ждёт человека. */
const QUICK_MS = 1500
const SIGN_MS = 10 * 60 * 1000
const POLL_MS = 4000

let present = false
let person: TokenPerson | null = null
let seq = 0
const waiters = new Map<number, (r: TokenAgentResponseV1) => void>()
let started = false

function onMessage(ev: MessageEvent): void {
  if (ev.source !== window || ev.origin !== window.location.origin) return
  const d = ev.data as { source?: string; kind?: string; id?: number; response?: TokenAgentResponseV1 } | null
  if (!d || d.source !== EXT) return
  if (d.kind === 'present') {
    present = true
    if (status.value === 'agent_missing') void refreshTokenStatus()
    return
  }
  if (typeof d.id === 'number' && d.response) {
    const w = waiters.get(d.id)
    if (w) {
      waiters.delete(d.id)
      w(d.response)
    }
  }
}

/** Идентификатор запроса протокола (UUIDv4). */
function requestId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  const h = Array.from({ length: 32 }, () => Math.floor(Math.random() * 16).toString(16))
  h[12] = '4'
  h[16] = ((parseInt(h[16] ?? '0', 16) & 3) | 8).toString(16)
  const s = h.join('')
  return `${s.slice(0, 8)}-${s.slice(8, 12)}-${s.slice(12, 16)}-${s.slice(16, 20)}-${s.slice(20)}`
}

/** Ответ «ошибка» протокола. */
function errorReply(rid: string, code: string, message: string): TokenAgentResponseV1 {
  return { protocol_version: 1, request_id: rid, type: 'error', error: { code, message } }
}

/**
 * Запрос расширению. Нет ответа за `timeoutMs` — `signing.agent_not_found`.
 * @param body — вид запроса и его блок (без служебных полей)
 * @param timeoutMs — сколько ждать
 */
export function extensionRequest(
  body: Pick<TokenAgentRequestV1, 'type'> & Partial<Pick<TokenAgentRequestV1, 'sign' | 'sign_batch' | 'local_journal'>>,
  timeoutMs = QUICK_MS,
): Promise<TokenAgentResponseV1> {
  startTokenAgent()
  const request: TokenAgentRequestV1 = { protocol_version: 1, request_id: requestId(), origin: window.location.origin, ...body }
  const id = ++seq
  return new Promise((resolve) => {
    const timer = setTimeout(() => {
      waiters.delete(id)
      resolve(errorReply(request.request_id, 'signing.agent_not_found', 'расширение «Главный — подпись» не ответило'))
    }, timeoutMs)
    waiters.set(id, (r) => {
      clearTimeout(timer)
      resolve(r)
    })
    window.postMessage({ source: PAGE, id, request, person }, window.location.origin)
  })
}

/** Время ожидания подписи: человек читает сводку и вводит PIN. */
export const SIGN_TIMEOUT_MS = SIGN_MS

/** Промахов подряд у расширения, которое уже отвечало. */
let misses = 0

/**
 * Перечитать состояние ключа у расширения. Расширение, которое уже отвечало,
 * может не успеть за 1,5 с (занято окном подписи, служба просыпается) —
 * «расширения нет» только после двух промахов подряд, иначе прежнее состояние.
 */
export async function refreshTokenStatus(): Promise<TokenStatus> {
  const r = await extensionRequest({ type: 'status' })
  if (r.type !== 'status' || !r.status) {
    misses += 1
    if (present && misses < 2 && status.value !== 'agent_missing') return status.value
    status.value = 'agent_missing'
    info.value = null
    return status.value
  }
  misses = 0
  present = true
  const s = r.status as TokenInfo
  info.value = s
  status.value = !s.token_present ? 'missing' : s.pin_unlocked ? 'inserted' : 'locked'
  return status.value
}

/**
 * Сообщить расширению, кто вошёл: подпись и индикатор «Токен» — по его ключу.
 * Смена человека — состояние перечитывается сразу.
 * @param p — вошедший (псевдоним и имя) или null
 */
export function setTokenPerson(p: TokenPerson | null): void {
  const next = p?.id ? { id: p.id, name: p.name } : null
  if (next?.id === person?.id && next?.name === person?.name) return
  person = next
  if (started) void refreshTokenStatus()
}

/** Запустить мост: слушать расширение и опрашивать состояние (раз в 4 с и при возврате на вкладку). */
export function startTokenAgent(): void {
  if (started || typeof window === 'undefined') return
  started = true
  window.addEventListener('message', onMessage)
  window.postMessage({ source: PAGE, kind: 'ping' }, window.location.origin)
  void refreshTokenStatus()
  setInterval(() => {
    if (present && document.visibilityState === 'visible') void refreshTokenStatus()
  }, POLL_MS)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') void refreshTokenStatus()
  })
}

/** Текущее состояние токена (только чтение). */
export function useTokenStatus(): Readonly<Ref<TokenStatus>> {
  startTokenAgent()
  return readonly(status)
}

/** Сведения о ключе: владелец, ключи, класс хранения (только чтение). */
export function useTokenInfo(): Readonly<Ref<TokenInfo | null>> {
  startTokenAgent()
  return readonly(info) as Readonly<Ref<TokenInfo | null>>
}
