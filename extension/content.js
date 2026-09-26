// Мост страницы системы «Главный» и расширения (AD-14): страница шлёт
// window.postMessage({source: 'glavny-page', id, request, person}), ответ приходит
// {source: 'glavny-ext', id, response}. person — {id, name} вошедшего в
// «Главный»: расширение подписывает его ключом (ключи многих персон, Д-72). Ключей и PIN здесь нет — только
// пересылка; origin страницы проверяет фоновая служба.
'use strict'

/* global chrome */
;(() => {
  if (window.__glavnyBridge) return
  window.__glavnyBridge = true
  const version = chrome.runtime.getManifest().version
  // Запросы, ответ на которые придёт из окна подтверждения: request_id → id страницы.
  const waiting = new Map()

  const post = (id, response) => window.postMessage({ source: 'glavny-ext', id, response }, window.location.origin)

  window.addEventListener('message', (ev) => {
    if (ev.source !== window || ev.origin !== window.location.origin) return
    const d = ev.data
    if (!d || d.source !== 'glavny-page') return
    if (d.kind === 'ping') {
      window.postMessage({ source: 'glavny-ext', kind: 'present', version }, window.location.origin)
      return
    }
    const rq = d.request
    chrome.runtime.sendMessage({ kind: 'page', request: rq, person: d.person }, (res) => {
      if (chrome.runtime.lastError) {
        post(d.id, {
          protocol_version: 1,
          request_id: rq?.request_id,
          type: 'error',
          error: { code: 'signing.agent_not_found', message: chrome.runtime.lastError.message },
        })
        return
      }
      if (res && res.pending) {
        waiting.set(res.pending, d.id)
        return
      }
      post(d.id, res)
    })
  })

  chrome.runtime.onMessage.addListener((msg) => {
    if (msg?.kind !== 'result') return
    const id = waiting.get(msg.rid)
    if (id === undefined) return
    waiting.delete(msg.rid)
    post(id, msg.response)
  })

  window.postMessage({ source: 'glavny-ext', kind: 'present', version }, window.location.origin)
})()
