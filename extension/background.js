// Фоновая служба расширения «Главный — подпись» (FR-69, AD-14, Д-72).
//
// Страница системы → content.js → сюда. Два адаптера за одним протоколом
// (contracts/internal/token-agent):
//   - ключ в браузере (по умолчанию): ключ под PIN в хранилище расширения,
//     подпись — пакетом подписи WASM; уровень 2 — окно подтверждения
//     confirm.html (страница расширения, её origin сервер не контролирует);
//   - физический ключ: запрос уходит агенту токена по Native Messaging,
//     окно и PIN — в локальной программе.
// Отказ уровня 1 вне перечня, отпечаток и сводку считает пакет подписи сам.
'use strict'

importScripts('wasm_exec.js', 'common.js')

/* global GLAVNY, glavnyStore, glavnyCall, glavnyReply, glavnyError, glavnyBrowserStatus, glavnyNow, glavnyKeys, glavnyPick, glavnyPersonText, chrome */

// ---------------------------------------------------------------- допуск ---

async function originAllowed(origin) {
  if (GLAVNY.STATIC_ORIGINS.some((re) => re.test(origin))) return true
  const s = await glavnyStore.settings()
  return s.origins.includes(origin)
}

// ------------------------------------------------------- физический ключ ---

let nativePort = null
const nativeWaiters = new Map()

function native() {
  if (nativePort) return nativePort
  nativePort = chrome.runtime.connectNative(GLAVNY.HOST)
  nativePort.onMessage.addListener((msg) => {
    const w = nativeWaiters.get(msg.request_id)
    if (w) {
      nativeWaiters.delete(msg.request_id)
      w(msg)
    }
  })
  nativePort.onDisconnect.addListener(() => {
    const err = chrome.runtime.lastError?.message || 'агент токена отключился'
    for (const [id, w] of nativeWaiters) w(glavnyError(id, 'signing.agent_not_found', err))
    nativeWaiters.clear()
    nativePort = null
  })
  return nativePort
}

function askAgent(request) {
  return new Promise((resolve) => {
    try {
      nativeWaiters.set(request.request_id, resolve)
      native().postMessage(request)
    } catch (e) {
      nativeWaiters.delete(request.request_id)
      resolve(glavnyError(request.request_id, 'signing.agent_not_found', e.message))
    }
  })
}

// ------------------------------------------------------ ключ в браузере ---

async function callContext() {
  const s = await glavnyStore.settings()
  const { seen_checkpoint: seen } = await glavnyStore.local('seen_checkpoint')
  return { now: glavnyNow(), workplace_id: s.workplace_id || '', seen_checkpoint: seen || 0, profile: s.profile || 'gost' }
}

/** Подписать в фоне (уровень 1 при разблокированном ключе — без окна). set — хранилища ключей вошедшего. */
async function signQuiet(rq, block, set, dk, ctx) {
  const { journal = [] } = await glavnyStore.local('journal')
  const r = await glavnyCall('sign', { block, sealed: set, dk_b64: dk, context: ctx, confirmed_digest: '', journal })
  if (!r.ok) return glavnyError(rq.request_id, r.code, r.message)
  await glavnyStore.setLocal({ journal: r.journal })
  return glavnyReply(rq.request_id, 'signed', { signed: [signedElem(r)] })
}

function signedElem(r) {
  return {
    envelope: r.envelope,
    doc_digest: r.prepared.doc_digest,
    summary: r.prepared.summary,
    local_journal_seq: r.entry.seq,
    client_signed_at: r.prepared.client_signed_at,
    key_storage: r.key_storage,
  }
}

/** Открытые сведения ключа для окна подтверждения (без хранилища). */
function keyEntryInfo(e) {
  return { key_ref: e.key_ref, person_id: e.person_id, profile: e.profile, key_storage: e.key_storage, storage_variant: e.storage_variant }
}

/** Открыть окно подтверждения; результат окно пришлёт вкладке само. */
async function openWindow(rq, blocks, prepared, ctx, tabId, origin, pick) {
  const rid = rq.request_id
  const w = await chrome.windows.create({
    url: chrome.runtime.getURL('confirm.html?rid=' + encodeURIComponent(rid)),
    type: 'popup',
    width: 560,
    height: 720,
    focused: true,
  })
  await glavnyStore.setSession({
    ['pending:' + rid]: { rq, blocks, prepared, ctx, tabId, origin, windowId: w.id, person: pick.person, set: pick.set, entries: pick.entries.map(keyEntryInfo) },
  })
  return { pending: rid }
}

/**
 * Подпись ключом в браузере. Ключ — того, кто вошёл в «Главный» (person со
 * страницы), профиль — из настроек (gost по умолчанию), key_ref запроса — в
 * приоритете. Нет ключа этого человека — отказ «Ключ ‹имя› не загружен».
 */
async function signBrowser(rq, tabId, origin, person) {
  const blocks = rq.type === 'sign_batch' ? rq.sign_batch || [] : rq.sign ? [rq.sign] : []
  if (!blocks.length) return glavnyError(rq.request_id, 'api.validation_failed', 'нет блока sign')
  const { names = {} } = await glavnyStore.local('names')
  const pick = glavnyPick(await glavnyKeys(), person?.id, blocks[0].key_ref)
  if (pick.error) {
    const msg = pick.missing ? `Ключ ${glavnyPersonText(pick.person, names)} не загружен — «Главный — подпись» → «Управление ключом» → «Загрузить ключи»` : pick.error
    return glavnyError(rq.request_id, 'signing.token_missing', msg)
  }
  const set = pick.set
  const ctx = await callContext()
  for (const b of blocks) {
    const cp = b.command_request?.seen_checkpoint
    if (cp && cp > ctx.seen_checkpoint) {
      ctx.seen_checkpoint = cp
      await glavnyStore.setLocal({ seen_checkpoint: cp })
    }
  }
  // Пакет подписи сам собирает подписываемое, считает отпечаток и сводку и
  // отклоняет уровень 1 вне перечня — до всякого окна.
  const prepared = []
  for (const b of blocks) {
    const r = await glavnyCall('prepare', b, set, ctx)
    if (!r.ok) return glavnyError(rq.request_id, r.code, r.message)
    prepared.push(r.prepared)
  }
  const { dk_b64: dk } = await glavnyStore.session('dk_b64')
  if (blocks.length === 1 && blocks[0].level === 1 && dk) return signQuiet(rq, blocks[0], set, dk, ctx)
  return openWindow(rq, blocks, prepared, ctx, tabId, origin, pick)
}

// ------------------------------------------------------------- запросы ---

/** Вошедший в «Главный» человек из сообщения страницы: {id, name} или null. */
function pagePerson(p) {
  const id = typeof p?.id === 'string' ? p.id.trim().toUpperCase() : ''
  if (!/^[A-Z0-9][A-Z0-9._-]{0,63}$/.test(id)) return null
  const name = typeof p.name === 'string' ? p.name.slice(0, 128) : ''
  return { id, name }
}

/** Запомнить имя человека со страницы — для списка ключей в «Управлении ключом». */
async function rememberName(person) {
  if (!person?.name) return
  const { names = {} } = await glavnyStore.local('names')
  if (names[person.id] !== person.name) await glavnyStore.setLocal({ names: { ...names, [person.id]: person.name } })
}

async function handlePage(rq, sender, rawPerson) {
  const origin = sender.origin || new URL(sender.url || 'about:blank').origin
  if (!(await originAllowed(origin))) return glavnyError(rq?.request_id, 'access.forbidden', 'адрес ' + origin + ' не разрешён в расширении')
  if (!rq || rq.protocol_version !== GLAVNY.PROTOCOL) return glavnyError(rq?.request_id, 'api.validation_failed', 'версия протокола')
  rq.origin = origin
  const person = pagePerson(rawPerson)
  await rememberName(person)
  const s = await glavnyStore.settings()
  if (s.adapter === 'agent') {
    const res = await askAgent(rq)
    if (res.type === 'status' && res.status) res.status.adapter = 'agent'
    return res
  }
  switch (rq.type) {
    case 'hello': {
      const v = await glavnyCall('version')
      const keys = Object.values(await glavnyKeys()).filter((e) => !person || e.person_id === person.id)
      return glavnyReply(rq.request_id, 'hello', {
        hello: {
          agent_version: v.agent_version,
          build_digest: 'streebog256:' + '0'.repeat(64),
          doc_format_versions: v.doc_format_versions,
          keys: keys.map((e) => ({ key_ref: e.key_ref, person_id: e.person_id, profile: e.profile, key_storage: e.key_storage })),
        },
      })
    }
    case 'status':
      return glavnyReply(rq.request_id, 'status', { status: { ...(await glavnyBrowserStatus(person?.id)), adapter: 'browser' } })
    case 'sign':
    case 'sign_batch':
      return signBrowser(rq, sender.tab?.id, origin, person)
    case 'local_journal': {
      const { journal = [] } = await glavnyStore.local('journal')
      const since = rq.local_journal?.since_seq || 0
      return glavnyReply(rq.request_id, 'local_journal', { local_journal: journal.filter((e) => e.seq > since) })
    }
    default:
      return glavnyError(rq.request_id, 'api.validation_failed', 'вид запроса ' + rq.type)
  }
}

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg?.kind === 'page' && sender.tab) {
    handlePage(msg.request, sender, msg.person)
      .then(sendResponse)
      .catch((e) => sendResponse(glavnyError(msg.request?.request_id, 'signing.agent_not_found', e.message)))
    return true
  }
  if (msg?.kind === 'lock') {
    glavnyStore.lock().then(() => sendResponse({ ok: true }))
    return true
  }
  if (msg?.kind === 'agent' && sender.id === chrome.runtime.id) {
    askAgent({ protocol_version: GLAVNY.PROTOCOL, request_id: crypto.randomUUID(), origin: 'chrome-extension://' + chrome.runtime.id, ...msg.request }).then(sendResponse)
    return true
  }
  return false
})

// Окно закрыли без решения — вкладке отказ.
chrome.windows.onRemoved.addListener(async (windowId) => {
  const all = await chrome.storage.session.get(null)
  for (const [k, p] of Object.entries(all)) {
    if (k.startsWith('pending:') && p.windowId === windowId) {
      await chrome.storage.session.remove(k)
      if (p.tabId != null) {
        chrome.tabs.sendMessage(p.tabId, { kind: 'result', rid: p.rq.request_id, response: glavnyError(p.rq.request_id, 'signing.cancelled', 'окно подтверждения закрыто') }).catch(() => {})
      }
    }
  }
})

// Блокировка экрана и конец смены — ключ снова под PIN (AD-14).
chrome.idle.setDetectionInterval(15 * 60)
chrome.idle.onStateChanged.addListener((state) => {
  if (state === 'locked') glavnyStore.lock()
})
chrome.alarms.onAlarm.addListener((a) => {
  if (a.name === 'autolock') glavnyStore.lock()
})
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === 'session' && changes.dk_b64?.newValue) chrome.alarms.create('autolock', { delayInMinutes: GLAVNY.AUTOLOCK_MIN })
})

// Хранилище сеанса — только страницам расширения, не content-скриптам.
chrome.storage.session.setAccessLevel?.({ accessLevel: 'TRUSTED_CONTEXTS' })

// Разрешённые пользователем адреса — content.js на них (регистрация при старте).
async function registerOrigins() {
  const s = await glavnyStore.settings()
  const matches = s.origins.map((o) => o + '/*')
  try {
    await chrome.scripting.unregisterContentScripts({ ids: ['glavny-origins'] }).catch(() => {})
    if (matches.length) {
      await chrome.scripting.registerContentScripts([{ id: 'glavny-origins', matches, js: ['content.js'], runAt: 'document_start', persistAcrossSessions: true }])
    }
  } catch (e) {
    console.warn('Главный: адреса не зарегистрированы', e)
  }
}
chrome.runtime.onInstalled.addListener(registerOrigins)
// Прежняя версия держала один ключ (запись sealed) под своей солью — ключи
// загружаются заново разом, под один PIN.
chrome.runtime.onInstalled.addListener(() => chrome.storage.local.remove('sealed'))
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === 'local' && changes.settings) registerOrigins()
})
