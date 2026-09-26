// Управление ключом (AD-14, Д-72): загрузка ключа персоны файлом и
// шифрование под PIN, блокировка и удаление, выбор адаптера (ключ в
// браузере / физический ключ), рабочее место, локальный журнал.
'use strict'

/* global glavnyStore, glavnyCall, glavnyNow, glavnyBrowserStatus, glavnyStorageText, glavnyErrorText, chrome */

const $ = (id) => document.getElementById(id)
let files = []

function text(el, s, cls) {
  el.textContent = s
  el.className = cls || ''
}

async function renderStatus() {
  const st = await glavnyBrowserStatus()
  const box = $('key-status')
  box.textContent = ''
  if (!st.token_present) {
    box.innerHTML = '<span class="badge">ключ не загружен</span> <span class="muted">— загрузите файл ключа ниже</span>'
    return
  }
  const { sealed } = await glavnyStore.local('sealed')
  const badge = document.createElement('span')
  badge.className = 'badge' + (st.pin_unlocked ? ' ready' : '')
  badge.textContent = st.pin_unlocked ? 'готов — PIN введён' : 'заблокирован — PIN спросит окно подписи'
  const p = document.createElement('p')
  p.textContent = `${sealed.person_id} · ${glavnyStorageText(sealed.key_storage, sealed.storage_variant)} · загружен ${sealed.sealed_at || ''}`
  const ul = document.createElement('ul')
  for (const k of sealed.keys) {
    const li = document.createElement('li')
    li.innerHTML = `<code></code> — ${k.profile}, отпечаток <code></code>`
    li.querySelectorAll('code')[0].textContent = k.key_ref
    li.querySelectorAll('code')[1].textContent = k.fingerprint.replace('streebog256:', '').slice(0, 16)
    ul.append(li)
  }
  box.append(badge, p, ul)
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
  for (const f of ev.target.files) {
    const t = await f.text()
    const r = await glavnyCall('inspectKey', t)
    if (!r.ok) {
      info.push(`${f.name}: ${glavnyErrorText(r)}`)
      continue
    }
    files.push(t)
    info.push(`${r.key_ref} — ${r.profile}, владелец ${r.person_id}`)
  }
  $('files-info').textContent = info.join('; ')
})

$('load').addEventListener('click', async () => {
  const msg = $('load-msg')
  if (!files.length) return text(msg, 'Выберите файл ключа', 'error')
  if ($('pin1').value !== $('pin2').value) return text(msg, 'PIN не совпадает', 'error')
  text(msg, 'Шифрую ключ под PIN…', 'muted')
  await new Promise((r) => setTimeout(r, 30))
  const r = await glavnyCall('seal', { files, key_storage: 'software_browser', storage_variant: 'extension', now: glavnyNow() }, $('pin1').value)
  if (!r.ok) return text(msg, glavnyErrorText(r), 'error')
  await glavnyStore.setLocal({ sealed: r.sealed })
  // Сразу разблокировать тем же PIN — индикатор «Токен» в шапке станет «готов».
  const u = await glavnyCall('unlock', r.sealed, $('pin1').value)
  if (u.ok) await glavnyStore.setSession({ dk_b64: u.dk_b64, unlocked_at: new Date().toISOString() })
  $('pin1').value = $('pin2').value = ''
  $('files').value = ''
  files = []
  text(msg, `Ключ ${r.sealed.person_id} загружен и разблокирован. Файл ключа больше не нужен браузеру.`, 'ok')
  renderStatus()
})

$('lock').addEventListener('click', async () => {
  await glavnyStore.lock()
  renderStatus()
})

$('remove').addEventListener('click', async () => {
  if (!confirm('Удалить ключ из браузера? Подписывать этим ключом здесь будет нельзя, пока не загрузите файл снова.')) return
  await glavnyStore.lock()
  await chrome.storage.local.remove('sealed')
  renderStatus()
})

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
  const { sealed } = await glavnyStore.local('sealed')
  if (!sealed) return text($('level1-msg'), 'Сначала загрузите ключ', 'error')
  const body = { command_id: crypto.randomUUID(), basis_seq: 1, policy_seq: 1, verdict: 'pass' }
  const block = {
    level: 1,
    payload_type: 'application/vnd.ant.event+json; v=1',
    payload_b64: btoa(unescape(encodeURIComponent(JSON.stringify(body)))),
    event_type: 'inspection.result.recorded',
    command_request: { operation: 'quality.inspection.record', item_id: 'ITEM-DEMO' },
  }
  const r = await glavnyCall('prepare', block, sealed, { now: glavnyNow() })
  text($('level1-msg'), r.ok ? 'Ошибка: уровень 1 не отклонён!' : 'Отклонено пакетом подписи — ' + glavnyErrorText(r), r.ok ? 'error' : 'ok')
})

$('shift').addEventListener('click', async () => {
  const { journal = [] } = await glavnyStore.local('journal')
  const { sealed } = await glavnyStore.local('sealed')
  const to = new Date()
  const from = new Date(to.getTime() - 24 * 3600 * 1000)
  const iso = (d) => d.toISOString().replace(/\.(\d{3})\d*Z$/, '.$1Z')
  const base = {
    format_version: 1,
    crypto_profile: 'gost',
    signers: sealed ? [sealed.keys[0].key_ref] : ['none@1'],
    person_id: sealed?.person_id || '—',
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
