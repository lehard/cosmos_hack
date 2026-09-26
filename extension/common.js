// Общее для фоновой службы и страниц расширения «Главный — подпись»
// (FR-69, AD-14, Д-72): хранилище, загрузка пакета подписи WASM, протокол.
// Пакет подписи — тот же Go-код, что у сервера (cmd/token-agent/internal/agent),
// собранный в WebAssembly: ГОСТ Р 34.10-2012 и ML-DSA в WebCrypto нет.
'use strict'

/* global Go, chrome */

const GLAVNY = {
  PROTOCOL: 1,
  HOST: 'ru.glavny.token_agent',
  STORAGE: 'software_browser',
  VARIANT: 'extension',
  // Адреса системы, где работает расширение без дополнительного разрешения.
  STATIC_ORIGINS: [/^http:\/\/127\.0\.0\.1(:\d+)?$/, /^http:\/\/localhost(:\d+)?$/],
  // Автоблокировка ключа (PIN снова) — через 8 часов (смена) или при блокировке экрана.
  AUTOLOCK_MIN: 8 * 60,
}

/** Время протокола: миллисекунды, UTC. */
function glavnyNow() {
  return new Date().toISOString().replace(/\.(\d{3})\d*Z$/, '.$1Z')
}

/** Идентификатор запроса (UUIDv4). */
function glavnyUUID() {
  return crypto.randomUUID()
}

const glavnyStore = {
  async local(keys) {
    return chrome.storage.local.get(keys)
  },
  async setLocal(obj) {
    return chrome.storage.local.set(obj)
  },
  async session(keys) {
    return chrome.storage.session.get(keys)
  },
  async setSession(obj) {
    return chrome.storage.session.set(obj)
  },
  async settings() {
    const { settings } = await chrome.storage.local.get('settings')
    return { adapter: 'browser', workplace_id: '', profile: 'gost', origins: [], ...(settings || {}) }
  },
  async lock() {
    await chrome.storage.session.remove(['dk_b64', 'unlocked_at'])
  },
}

let glavnySignerPromise = null

/** Пакет подписи WASM (globalThis.glavnySigner); загружается один раз. */
function glavnySigner() {
  if (!glavnySignerPromise) {
    glavnySignerPromise = new Promise((resolve, reject) => {
      globalThis.__glavnySignerReady = () => resolve(globalThis.glavnySigner)
      const go = new Go()
      fetch(chrome.runtime.getURL('signer.wasm'))
        .then((r) => r.arrayBuffer())
        .then((b) => WebAssembly.instantiate(b, go.importObject))
        .then((r) => {
          go.run(r.instance)
        })
        .catch(reject)
    })
  }
  return glavnySignerPromise
}

/** Вызов операции пакета: ответ JSON {ok, …} разобран. */
async function glavnyCall(name, ...args) {
  const s = await glavnySigner()
  return JSON.parse(s[name](...args.map((a) => (typeof a === 'string' ? a : JSON.stringify(a)))))
}

/** Ответ протокола агента (contracts/internal/token-agent/nm-response.v1.json). */
function glavnyReply(requestId, type, body) {
  return { protocol_version: GLAVNY.PROTOCOL, request_id: requestId, type, ...body }
}

function glavnyError(requestId, code, message) {
  return glavnyReply(requestId, 'error', { error: { code, message: String(message || code) } })
}

/** Состояние ключа в браузере: загружен ли, разблокирован ли. */
async function glavnyBrowserStatus() {
  const { sealed } = await glavnyStore.local('sealed')
  const { dk_b64: dk } = await glavnyStore.session('dk_b64')
  const settings = await glavnyStore.settings()
  const status = { token_present: !!sealed, pin_unlocked: !!(sealed && dk) }
  if (settings.workplace_id) status.workplace_id = settings.workplace_id
  if (sealed) {
    status.person_id = sealed.person_id
    status.key_storage = sealed.key_storage || GLAVNY.STORAGE
    status.storage_variant = sealed.storage_variant || GLAVNY.VARIANT
    status.key_refs = (sealed.keys || []).map((k) => k.key_ref)
  }
  return status
}

/** Класс хранения ключа словами (как в отчёте верификатора). */
function glavnyStorageText(storage, variant) {
  if (storage === 'hardware_token') return 'физический ключ'
  if (storage === 'software_browser') return variant === 'page' ? 'ключ в браузере (хранилище страницы)' : 'ключ в браузере (расширение)'
  return '—'
}

/** Текст ошибки пакета подписи для человека. */
function glavnyErrorText(r) {
  const known = {
    'signing.pin_wrong': 'Неверный PIN',
    'signing.level_not_allowed': 'Уровень подписи не допускается',
    'signing.token_missing': 'Ключ не загружен',
    'signing.document_changed': 'Подписываемое изменилось',
    'signing.rate_limited': 'Слишком часто',
    'signing.cancelled': 'Подпись отменена',
  }
  return (known[r.code] ? known[r.code] + ': ' : '') + (r.message || r.code)
}
