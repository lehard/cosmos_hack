// Окно подтверждения уровня 2 (AD-13, AD-14): показывает сводку, которую
// пакет подписи посчитал сам из подписываемых байтов, спрашивает PIN, если
// ключ заблокирован, и подписывает здесь же. Подписывается ровно то, что
// показано: пакет подписи сверяет отпечаток с подтверждённым.
'use strict'

/* global glavnyStore, glavnyCall, glavnyError, glavnyReply, glavnyStorageText, glavnyErrorText, chrome */

const rid = new URLSearchParams(location.search).get('rid')
const $ = (id) => document.getElementById(id)
let pending = null
let sealed = null

function field(label, value) {
  const dt = document.createElement('dt')
  dt.textContent = label
  const dd = document.createElement('dd')
  dd.textContent = value
  $('summary').append(dt, dd)
}

async function finish(response) {
  await chrome.storage.session.remove('pending:' + rid)
  if (pending?.tabId != null) {
    await chrome.tabs.sendMessage(pending.tabId, { kind: 'result', rid, response }).catch(() => {})
  }
  window.close()
}

async function init() {
  pending = (await glavnyStore.session('pending:' + rid))['pending:' + rid]
  sealed = (await glavnyStore.local('sealed')).sealed
  if (!pending || !sealed) {
    $('error').textContent = 'Запрос подписи не найден или ключ не загружен — закройте окно.'
    $('sign').disabled = true
    return
  }
  const { prepared, blocks, origin } = pending
  $('origin').textContent = 'Запрос со страницы ' + origin
  if (blocks.length > 1) {
    $('title').textContent = `Подписать пачку: ${blocks.length} изделий, одно решение`
    field('Изделия', prepared.map((p) => p.item_id || '—').join(', '))
  }
  for (const f of prepared[0].summary) field(f.label, f.value)
  $('key').textContent = `${sealed.person_id} · ${prepared[0].signers.join(' + ')} · ${glavnyStorageText(sealed.key_storage, sealed.storage_variant)}`
  const { dk_b64: dk } = await glavnyStore.session('dk_b64')
  $('pin-box').hidden = !!dk
  if (!dk) $('pin').focus()
  else $('sign').focus()
}

async function sign() {
  $('error').textContent = ''
  $('sign').disabled = true
  $('sign').textContent = 'Подписываю…'
  // Отрисовать «Подписываю…» до долгого argon2id.
  await new Promise((r) => setTimeout(r, 30))
  try {
    let { dk_b64: dk } = await glavnyStore.session('dk_b64')
    if (!dk) {
      const u = await glavnyCall('unlock', sealed, $('pin').value)
      if (!u.ok) throw u
      dk = u.dk_b64
      await glavnyStore.setSession({ dk_b64: dk, unlocked_at: new Date().toISOString() })
    }
    let { journal = [] } = await glavnyStore.local('journal')
    const signed = []
    for (let i = 0; i < pending.blocks.length; i++) {
      const r = await glavnyCall('sign', {
        block: pending.blocks[i],
        sealed,
        dk_b64: dk,
        context: pending.ctx,
        confirmed_digest: pending.prepared[i].doc_digest,
        journal,
      })
      if (!r.ok) throw r
      journal = r.journal
      signed.push({
        envelope: r.envelope,
        doc_digest: r.prepared.doc_digest,
        summary: r.prepared.summary,
        local_journal_seq: r.entry.seq,
        client_signed_at: r.prepared.client_signed_at,
        key_storage: r.key_storage,
      })
    }
    await glavnyStore.setLocal({ journal })
    await finish(glavnyReply(rid, 'signed', { signed }))
  } catch (e) {
    const r = e && e.code ? e : { code: 'signing.agent_not_found', message: String(e?.message || e) }
    $('error').textContent = glavnyErrorText(r)
    $('sign').disabled = false
    $('sign').textContent = 'Подписать'
    if (r.code === 'signing.pin_wrong') {
      $('pin').value = ''
      $('pin').focus()
    }
  }
}

$('sign').addEventListener('click', sign)
$('pin').addEventListener('keydown', (e) => e.key === 'Enter' && sign())
$('cancel').addEventListener('click', () => finish(glavnyError(rid, 'signing.cancelled', 'подписант отказался в окне подтверждения')))
init()
