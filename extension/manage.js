// Управление ключом (AD-14, Д-72): загрузка ключей персон файлами — много
// за раз, все под один PIN, — список ключей (владелец, роль, профиль, класс),
// удаление по одному и всех, блокировка, выбор адаптера (ключ в браузере /
// физический ключ), рабочее место, локальный журнал.
'use strict'

/* global glavnyStore, glavnyCall, glavnyNow, glavnyBrowserStatus, glavnyStorageText, glavnyErrorText, glavnyKeys, glavnyPersons, GLAVNY_ROLES, chrome */

const $ = (id) => document.getElementById(id)
let files = []

function text(el, s, cls) {
  el.textContent = s
  el.className = cls || ''
}

async function renderStatus() {
  const st = await glavnyBrowserStatus()
  const keys = await glavnyKeys()
  const { names = {} } = await glavnyStore.local('names')
  const box = $('key-status')
  box.textContent = ''
  const any = st.keys_total > 0
  // Ключи уже есть — новые ложатся под тот же PIN: второй раз вводить не нужно.
  $('pin2-box').hidden = any
  $('pin1-label').textContent = any ? 'PIN уже загруженных ключей' : 'PIN для всех ключей (не короче 4 символов)'
  $('keys-table').hidden = !any
  const tb = $('keys')
  tb.textContent = ''
  if (!any) {
    box.innerHTML = '<span class="badge">ключи не загружены</span> <span class="muted">— загрузите файлы ключей ниже</span>'
    return
  }
  const badge = document.createElement('span')
  badge.className = 'badge' + (st.pin_unlocked ? ' ready' : '')
  badge.textContent = st.pin_unlocked ? 'готовы — PIN введён' : 'заблокированы — PIN спросит окно подписи'
  const p = document.createElement('p')
  p.textContent = `Ключей: ${st.keys_total}, людей: ${st.persons.length}`
  box.append(badge, p)
  const order = { gost: 0, pq: 1 }
  const list = Object.values(keys).sort((a, b) => a.person_id.localeCompare(b.person_id) || (order[a.profile] ?? 9) - (order[b.profile] ?? 9))
  for (const e of list) {
    const tr = document.createElement('tr')
    const owner = names[e.person_id] ? `${names[e.person_id]} (${e.person_id})` : e.person_id
    const profile = e.profile === 'pq' ? 'pq — ML-DSA-65' : e.profile === 'gost' ? 'gost — ГОСТ Р 34.10-2012' : e.profile
    for (const v of [owner, GLAVNY_ROLES[e.person_id] || '—', profile, glavnyStorageText(e.key_storage, e.storage_variant)]) {
      const td = document.createElement('td')
      td.textContent = v
      tr.append(td)
    }
    const ref = document.createElement('td')
    const code = document.createElement('code')
    code.textContent = e.key_ref
    code.title = 'отпечаток ' + (e.fingerprint || '').replace('streebog256:', '').slice(0, 16)
    ref.append(code)
    const act = document.createElement('td')
    const del = document.createElement('button')
    del.type = 'button'
    del.className = 'danger'
    del.textContent = 'Удалить'
    del.addEventListener('click', () => removeKey(e.key_ref))
    act.append(del)
    tr.append(ref, act)
    tb.append(tr)
  }
}

/** Удалить один ключ; последний — и ключ сеанса из PIN. */
async function removeKey(keyRef) {
  if (!confirm(`Удалить ключ ${keyRef} из браузера?`)) return
  const keys = await glavnyKeys()
  delete keys[keyRef]
  await glavnyStore.setLocal({ keys })
  if (!Object.keys(keys).length) await glavnyStore.lock()
  renderStatus()
}

async function renderSettings() {
  const s = await glavnyStore.settings()
  for (const r of document.querySelectorAll('input[name=adapter]')) r.checked = r.value === s.adapter
  $('workplace').value = s.workplace_id || ''
  $('profile').value = s.profile || 'gost'
  $('origins').textContent = s.origins.length ? 'Разрешены: ' + s.origins.join(', ') : ''
}

async function renderJournal() {
  const { journal = [] } = await glavnyStore.local('journal')
  const tb = $('journal')
  tb.textContent = ''
  for (const e of journal.slice(-20).reverse()) {
    const tr = document.createElement('tr')
    for (const v of [e.seq, e.signed_at, e.event_type || '—', e.level, (e.doc_digest || e.digest).replace('streebog256:', '').slice(0, 16)]) {
      const td = document.createElement('td')
      td.textContent = String(v)
      tr.append(td)
    }
    tb.append(tr)
  }
}

$('files').addEventListener('change', async (ev) => {
  files = []
  const info = []
  const bad = []
  for (const f of ev.target.files) {
    const t = await f.text()
    const r = await glavnyCall('inspectKey', t)
    if (!r.ok) {
      bad.push(`${f.name}: ${glavnyErrorText(r)}`)
      continue
    }
    files.push(t)
    info.push(r)
  }
  const persons = [...new Set(info.map((r) => r.person_id))]
  const lines = [`Выбрано ключей: ${files.length} (${persons.length} чел.: ${persons.join(', ')})`, ...bad.map((b) => 'не принят — ' + b)]
  $('files-info').textContent = lines.join('; ')
})

$('load').addEventListener('click', async () => {
  const msg = $('load-msg')
  if (!files.length) return text(msg, 'Выберите файлы ключей', 'error')
  const keys = await glavnyKeys()
  const existing = Object.values(keys)[0]?.sealed || null
  if (!existing && $('pin1').value !== $('pin2').value) return text(msg, 'PIN не совпадает', 'error')
  text(msg, `Шифрую ключи (${files.length}) под PIN…`, 'muted')
  await new Promise((r) => setTimeout(r, 30))
  // argon2id — один раз: ключ из PIN, затем AES-256-GCM на каждый ключ.
  const now = glavnyNow()
  const r = await glavnyCall('sealEach', { files, key_storage: 'software_browser', storage_variant: 'extension', now, existing }, $('pin1').value)
  if (!r.ok) return text(msg, r.code === 'signing.pin_wrong' ? 'Неверный PIN: новые ключи ложатся под PIN уже загруженных (или удалите все ключи)' : glavnyErrorText(r), 'error')
  for (const sl of r.sealed) {
    const k = sl.keys[0]
    keys[k.key_ref] = {
      key_ref: k.key_ref,
      person_id: sl.person_id,
      profile: k.profile,
      fingerprint: k.fingerprint,
      key_storage: sl.key_storage,
      storage_variant: sl.storage_variant,
      loaded_at: now,
      sealed: sl,
    }
  }
  await glavnyStore.setLocal({ keys })
  // Сразу разблокировать: индикатор «Токен» в шапке станет «готов».
  await glavnyStore.setSession({ dk_b64: r.dk_b64, unlocked_at: new Date().toISOString() })
  $('pin1').value = $('pin2').value = ''
  $('files').value = ''
  $('files-info').textContent = ''
  files = []
  const persons = glavnyPersons(Object.fromEntries(r.sealed.map((sl) => [sl.keys[0].key_ref, sl])))
  text(msg, `Загружено ключей: ${r.sealed.length} (${persons.join(', ')}), разблокированы. Входите в «Главный» любой персоной — ключ выберется сам.`, 'ok')
  renderStatus()
})

$('lock').addEventListener('click', async () => {
  await glavnyStore.lock()
  renderStatus()
})

$('remove').addEventListener('click', async () => {
  if (!confirm('Удалить все ключи из браузера? Подписывать здесь будет нельзя, пока не загрузите файлы снова.')) return
  await glavnyStore.lock()
  await chrome.storage.local.remove(['keys', 'sealed'])
  renderStatus()
})

/** Хранилища для проверок на этой странице: ключи первого по алфавиту человека. */
async function anyPersonSet() {
  const keys = await glavnyKeys()
  const person = glavnyPersons(keys)[0]
  if (!person) return null
  const mine = Object.values(keys).filter((e) => e.person_id === person)
  const gost = mine.find((e) => e.profile === 'gost')
  return gost ? { person, set: [gost.sealed], keyRef: gost.key_ref } : null
}

$('save').addEventListener('click', async () => {
  const s = await glavnyStore.settings()
  s.adapter = document.querySelector('input[name=adapter]:checked')?.value || 'browser'
  s.workplace_id = $('workplace').value.trim()
  s.profile = $('profile').value
  await glavnyStore.setLocal({ settings: s })
  renderSettings()
})

$('origin-add').addEventListener('click', async () => {
  let o
  try {
    o = new URL($('origin').value.trim()).origin
  } catch {
    return
  }
  const granted = await chrome.permissions.request({ origins: [o + '/*'] })
  if (!granted) return
  const s = await glavnyStore.settings()
  if (!s.origins.includes(o)) s.origins.push(o)
  await glavnyStore.setLocal({ settings: s })
  $('origin').value = ''
  renderSettings()
})

$('agent-check').addEventListener('click', async () => {
  text($('agent-msg'), 'Спрашиваю агент токена…', 'muted')
  const r = await chrome.runtime.sendMessage({ kind: 'agent', request: { type: 'status' } })
  if (r?.type === 'status') {
    const s = r.status
    text(
      $('agent-msg'),
      s.token_present ? `Агент отвечает: токен ${s.person_id}, ${glavnyStorageText(s.key_storage)}, PIN ${s.pin_unlocked ? 'введён' : 'не введён'}` : 'Агент отвечает, токен не вставлен',
      'ok',
    )
  } else {
    text($('agent-msg'), 'Агент токена не найден: ' + (r?.error?.message || 'нет ответа') + ' — см. README, шаг «Физический ключ»', 'error')
  }
})

// «Годен» уровнем 1: пакет подписи отклоняет сам, до всякого окна.
$('level1').addEventListener('click', async () => {
  const pick = await anyPersonSet()
  if (!pick) return text($('level1-msg'), 'Сначала загрузите ключи', 'error')
  const body = { command_id: crypto.randomUUID(), basis_seq: 1, policy_seq: 1, verdict: 'pass' }
  const block = {
    level: 1,
    payload_type: 'application/vnd.ant.event+json; v=1',
    payload_b64: btoa(unescape(encodeURIComponent(JSON.stringify(body)))),
    event_type: 'inspection.result.recorded',
    command_request: { operation: 'quality.inspection.record', item_id: 'ITEM-DEMO' },
  }
  const r = await glavnyCall('prepare', block, pick.set, { now: glavnyNow() })
  text($('level1-msg'), r.ok ? 'Ошибка: уровень 1 не отклонён!' : 'Отклонено пакетом подписи — ' + glavnyErrorText(r), r.ok ? 'error' : 'ok')
})

$('shift').addEventListener('click', async () => {
  const { journal = [] } = await glavnyStore.local('journal')
  const pick = await anyPersonSet()
  const to = new Date()
  const from = new Date(to.getTime() - 24 * 3600 * 1000)
  const iso = (d) => d.toISOString().replace(/\.(\d{3})\d*Z$/, '.$1Z')
  const base = {
    format_version: 1,
    crypto_profile: 'gost',
    signers: pick ? [pick.keyRef] : ['none@1'],
    person_id: pick?.person || '—',
    shift_id: 'local',
    window_from: iso(from),
    window_to: iso(to),
    merkle_root: '',
    leaf_count: 0,
    counts_by_type: {},
  }
  const r = await glavnyCall('shiftReport', journal, base)
  if (!r.ok) return text($('shift-msg'), glavnyErrorText(r), 'error')
  const counts = Object.entries(r.report.counts_by_type).map(([k, n]) => `${k}: ${n}`).join(', ')
  text($('shift-msg'), `подписей ${r.report.leaf_count}, корень ${r.report.merkle_root.replace('streebog256:', '').slice(0, 16)}… ${counts}`, 'muted')
})

chrome.storage.onChanged.addListener(() => {
  renderStatus()
  renderJournal()
})
renderStatus()
renderSettings()
renderJournal()
