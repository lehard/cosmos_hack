// Всплывающее окно значка: состояние ключа и переход к управлению (AD-14).
'use strict'

/* global glavnyBrowserStatus, glavnyStore, glavnyStorageText, chrome */

async function render() {
  const s = await glavnyStore.settings()
  const st = await glavnyBrowserStatus()
  const el = document.getElementById('status')
  if (s.adapter === 'agent') el.textContent = 'Физический ключ: подпись через агент токена token-agent.'
  else if (!st.token_present) el.textContent = 'Ключ не загружен.'
  else el.textContent = `${st.person_id} · ${glavnyStorageText(st.key_storage, st.storage_variant)} · ${st.pin_unlocked ? 'готов' : 'заблокирован (PIN спросит окно подписи)'}`
}
document.getElementById('manage').addEventListener('click', () => chrome.runtime.openOptionsPage())
document.getElementById('lock').addEventListener('click', async () => {
  await glavnyStore.lock()
  render()
})
render()
