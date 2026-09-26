// Всплывающее окно значка: состояние ключей и переход к управлению (AD-14).
'use strict'

/* global glavnyBrowserStatus, glavnyStore, chrome */

async function render() {
  const s = await glavnyStore.settings()
  const st = await glavnyBrowserStatus()
  const el = document.getElementById('status')
  if (s.adapter === 'agent') el.textContent = 'Физический ключ: подпись через агент токена token-agent.'
  else if (!st.token_present) el.textContent = 'Ключи не загружены.'
  else
    el.textContent = `Ключи в браузере: ${st.keys_total} (${st.persons.join(', ')}) · ${st.pin_unlocked ? 'готовы — PIN введён' : 'заблокированы (PIN спросит окно подписи)'}. Подпись — ключом того, кто вошёл в «Главный».`
}
document.getElementById('manage').addEventListener('click', () => chrome.runtime.openOptionsPage())
document.getElementById('lock').addEventListener('click', async () => {
  await glavnyStore.lock()
  render()
})
render()
