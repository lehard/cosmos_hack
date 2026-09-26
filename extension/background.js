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

/* global GLAVNY, glavnyStore, glavnyCall, glavnyReply, glavnyError, glavnyBrowserStatus, glavnyNow, chrome */

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

/** Подписать в фоне (уровень 1 при разблокированном ключе — без окна). */
async function signQuiet(rq, block, sealed, dk, ctx) {
  const { journal = [] } = await glavnyStore.local('journal')
  const r = await glavnyCall('sign', { block, sealed, dk_b64: dk, context: ctx, confirmed_digest: '', journal })
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

/** Открыть окно подтверждения; результат окно пришлёт вкладке само. */
async function openWindow(rq, blocks, prepared, ctx, tabId, origin) {
  const rid = rq.request_id
  const w = await chrome.windows.create({
    url: chrome.runtime.getURL('confirm.html?rid=' + encodeURIComponent(rid)),
    type: 'popup',
    width: 560,
    height: 720,
    focused: true,
  })
  await glavnyStore.setSession({ ['pending:' + rid]: { rq, blocks, prepared, ctx, tabId, origin, windowId: w.id } })
  return { pending: rid }
}

async function signBrowser(rq, tabId, origin) {
  const { sealed } = await glavnyStore.local('sealed')
  if (!sealed) return glavnyError(rq.request_id, 'signing.token_missing', 'Ключ не загружен: откройте «Главный — подпись» → «Управление ключом»')
  const blocks = rq.type === 'sign_batch' ? rq.sign_batch || [] : rq.sign ? [rq.sign] : []
  if (!blocks.length) return glavnyError(rq.request_id, 'api.validation_failed', 'нет блока sign')
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
    const r = await glavnyCall('prepare', b, sealed, ctx)
    if (!r.ok) return glavnyError(rq.request_id, r.code, r.message)
    prepared.push(r.prepared)
  }
  const { dk_b64: dk } = await glavnyStore.session('dk_b64')
  if (blocks.length === 1 && blocks[0].level === 1 && dk) return signQuiet(rq, blocks[0], sealed, dk, ctx)
  return openWindow(rq, blocks, prepared, ctx, tabId, origin)
}

// ------------------------------------------------------------- запросы ---

async function handlePage(rq, sender) {
  const origin = sender.origin || new URL(sender.url || 'about:blank').origin
  if (!(await originAllowed(origin))) return glavnyError(rq?.request_id, 'access.forbidden', 'адрес ' + origin + ' не разрешён в расширении')
  if (!rq || rq.protocol_version !== GLAVNY.PROTOCOL) return glavnyError(rq?.request_id, 'api.validation_failed', 'версия протокола')
  rq.origin = origin
  const s = await glavnyStore.settings()
  if (s.adapter === 'agent') {
    const res = await askAgent(rq)
    if (res.type === 'status' && res.status) res.status.adapter = 'agent'
    return res
  }
  switch (rq.type) {
    case 'hello': {
      const v = await glavnyCall('version')
      const st = await glavnyBrowserStatus()
      return glavnyReply(rq.request_id, 'hello', {
        hello: {
          agent_version: v.agent_version,
          build_digest: 'streebog256:' + '0'.repeat(64),
          doc_format_versions: v.doc_format_versions,
          keys: (st.key_refs || []).map((k) => ({ key_ref: k, person_id: st.person_id, profile: k.includes('-pq@') ? 'pq' : 'gost', key_storage: st.key_storage })),
        },
      })
    }
    case 'status':
      return glavnyReply(rq.request_id, 'status', { status: { ...(await glavnyBrowserStatus()), adapter: 'browser' } })
    case 'sign':
    case 'sign_batch':
      return signBrowser(rq, sender.tab?.id, origin)
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
    handlePage(msg.request, sender)
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
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === 'local' && changes.settings) registerOrigins()
})
