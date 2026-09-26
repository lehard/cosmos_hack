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

/** Пакет подписи WASM (globalThis.glavnySigner); загружается один раз. Имя загрузчика отличается от globalThis.glavnySigner: WASM кладёт туда свой объект и затёр бы функцию — второй вызов падал «glavnySigner is not a function». */
function loadGlavnySigner() {
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
  const s = await loadGlavnySigner()
  return JSON.parse(s[name](...args.map((a) => (typeof a === 'string' ? a : JSON.stringify(a)))))
}

/** Ответ протокола агента (contracts/internal/token-agent/nm-response.v1.json). */
function glavnyReply(requestId, type, body) {
  return { protocol_version: GLAVNY.PROTOCOL, request_id: requestId, type, ...body }
}

function glavnyError(requestId, code, message) {
  return glavnyReply(requestId, 'error', { error: { code, message: String(message || code) } })
}

// ------------------------------------------------ ключи многих персон ---
//
// Демо из одного браузера (Д-72): расширение держит ключи всех персон
// рабочего места — словарь по key_ref в chrome.storage.local.keys. Каждый ключ
// — своё хранилище пакета подписи (AES-256-GCM), все под одним PIN: argon2id
// один раз, общая соль, один ключ сеанса dk_b64 открывает любое. Подписывает
// ключ того, кто вошёл в «Главный» (страница сообщает person_id); ключом
// другого человека сервер подпись не примет — «суперключа» нет.

/** Роли демо-персон (подпись в списке ключей; имя страница сообщает сама). */
const GLAVNY_ROLES = {
  'INS-01': 'контролёр ОТК',
  'HQC-01': 'начальник ОТК',
  'FOR-WC': 'мастер сварочного цеха',
  'TEC-01': 'технолог',
  'PM-01': 'руководитель производства',
  'ADM-01': 'администратор безопасности',
  'AUD-01': 'аудитор ИБ',
}

/** Порядок профилей в подписи: ГОСТ, затем ML-DSA (как signersFor в Go). */
const GLAVNY_PROFILE_ORDER = { gost: 0, pq: 1 }

/** Ключи в браузере: key_ref → {key_ref, person_id, profile, fingerprint, key_storage, storage_variant, loaded_at, sealed}. */
async function glavnyKeys() {
  const { keys } = await glavnyStore.local('keys')
  return keys || {}
}

/** Люди, чьи ключи загружены (по алфавиту). */
function glavnyPersons(keys) {
  return [...new Set(Object.values(keys).map((e) => e.person_id))].sort()
}

/** Как назвать человека: имя со страницы, роль из таблицы демо-персон, псевдоним. */
function glavnyPersonText(personId, names) {
  const name = names?.[personId]
  const role = GLAVNY_ROLES[personId]
  return [name || personId, name ? personId : '', role].filter(Boolean).join(' · ')
}

/**
 * Хранилища для подписи: ключи человека person по одному на профиль (key_ref
 * из запроса — в приоритете, иначе загруженный последним). Нет person и
 * загружены ключи одного человека — его. Ответ {person, set, entries} или
 * {error} (missing — у этого человека нет ключа).
 */
function glavnyPick(keys, person, keyRef) {
  const all = Object.values(keys)
  if (!all.length) return { error: 'Ключи не загружены: откройте «Главный — подпись» → «Управление ключом» → «Загрузить ключи»' }
  if (!person && keyRef && keys[keyRef]) person = keys[keyRef].person_id
  if (!person) {
    const persons = glavnyPersons(keys)
    if (persons.length !== 1) return { error: 'Загружены ключи нескольких человек, а страница не сообщила, кто вошёл, — обновите страницу «Главный»' }
    person = persons[0]
  }
  if (keyRef && keys[keyRef] && keys[keyRef].person_id !== person) {
    return { person, error: `Ключ ${keyRef} принадлежит ${keys[keyRef].person_id}, а вошёл ${person} — подписывать за другого нельзя` }
  }
  const byProfile = {}
  for (const e of all) {
    if (e.person_id !== person) continue
    const cur = byProfile[e.profile]
    if (e.key_ref === keyRef || !cur || (cur.key_ref !== keyRef && (e.loaded_at || '') > (cur.loaded_at || ''))) byProfile[e.profile] = e
  }
  if (!byProfile.gost) return { person, missing: true, error: `Ключ ${person} не загружен` }
  const set = Object.values(byProfile)
    .sort((a, b) => (GLAVNY_PROFILE_ORDER[a.profile] ?? 9) - (GLAVNY_PROFILE_ORDER[b.profile] ?? 9))
    .map((e) => e.sealed)
  return { person, set, entries: Object.values(byProfile) }
}

/**
 * Состояние ключа в браузере для вошедшего человека: загружен ли его ключ,
 * разблокирован ли PIN. Без person — есть ли ключи вообще.
 */
async function glavnyBrowserStatus(person) {
  const keys = await glavnyKeys()
  const { dk_b64: dk } = await glavnyStore.session('dk_b64')
  const settings = await glavnyStore.settings()
  const all = Object.values(keys)
  const persons = glavnyPersons(keys)
  const who = person || (persons.length === 1 ? persons[0] : '')
  const mine = who ? all.filter((e) => e.person_id === who) : []
  const present = person ? mine.some((e) => e.profile === 'gost') : all.length > 0
  const status = { token_present: present, pin_unlocked: !!(present && dk), persons, keys_total: all.length }
  if (settings.workplace_id) status.workplace_id = settings.workplace_id
  if (who) status.person_id = who
  if (mine.length) {
    status.key_storage = mine[0].key_storage || GLAVNY.STORAGE
    status.storage_variant = mine[0].storage_variant || GLAVNY.VARIANT
    status.key_refs = mine.map((e) => e.key_ref)
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
    'signing.document_changed': 'Подписываемое изменилось',
    'signing.rate_limited': 'Слишком часто',
    'signing.cancelled': 'Подпись отменена',
  }
  return (known[r.code] ? known[r.code] + ': ' : '') + (r.message || r.code)
}
