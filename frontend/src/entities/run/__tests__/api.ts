// Подмена сети для тестов контейнеров эпика 14: сгенерированный клиент зовёт
// глобальный fetch — тест отвечает по «МЕТОД путь» заготовленным телом (или
// функцией от тела запроса) и запоминает запросы. Нет ключа — 501
// api.not_implemented, как ответит сервер на объявленную, но не наполненную операцию.
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { vi } from 'vitest'
import type { Component } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { i18n } from '@/shared/i18n'

export interface Call {
  method: string
  path: string
  query: URLSearchParams
  body: unknown
}

/** Ответ: тело или функция от тела запроса; `{ __status, __body }` — ошибка с кодом. */
export type Route = unknown | ((body: unknown) => unknown)

/** Ошибка problem+json для маршрута. */
export const problem = (status: number, code: string) => ({ __status: status, __body: { type: `urn:ant:problem:${code}`, title: code, status, code } })

/** Подменить fetch: ответы по ключу `GET /api/v1/…` (путь без параметров запроса). */
export function mockApi(routes: Record<string, Route>): Call[] {
  const calls: Call[] = []
  vi.stubGlobal('fetch', async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    const u = new URL(url, 'http://ant.local')
    const path = decodeURIComponent(u.pathname)
    const body = init?.body ? JSON.parse(String(init.body)) : undefined
    calls.push({ method, path, query: u.searchParams, body })
    const key = `${method} ${path}`
    const headers = { 'Content-Type': 'application/json', 'Ant-Backend': 'fixtures' }
    if (!(key in routes)) {
      const p = problem(501, 'api.not_implemented')
      return new Response(JSON.stringify(p.__body), { status: 501, headers: { ...headers, 'Content-Type': 'application/problem+json' } })
    }
    const r = routes[key]
    const out = typeof r === 'function' ? (r as (b: unknown) => unknown)(body) : r
    if (out && typeof out === 'object' && '__status' in out) {
      const e = out as { __status: number; __body: unknown }
      return new Response(JSON.stringify(e.__body), { status: e.__status, headers: { ...headers, 'Content-Type': 'application/problem+json' } })
    }
    return new Response(JSON.stringify(out), { status: 200, headers })
  })
  return calls
}

/** Дождаться ответов и перерисовки. */
export async function settle(): Promise<void> {
  for (let i = 0; i < 8; i++) {
    await flushPromises()
    await new Promise((r) => setTimeout(r, 0))
  }
}

/** Смонтировать контейнер виджета с Pinia, текстами, маршрутизатором и Vue Query. */
export async function mountWidget(component: Component, props: Record<string, unknown>, pinia: Pinia = createPinia()) {
  setActivePinia(pinia)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
  await router.push('/')
  const w = mount(component, {
    props: { slotId: 's', slice: {}, density: 'comfortable', ...props },
    global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] },
  })
  await settle()
  return w
}

/** Сеанс администратора: версия политики и рабочее место для команд (AD-39, AD-15). */
export const adminSession = {
  person_id: 'P-ADM-1',
  display_name: 'Админ А.',
  roles: [{ role_id: 'administrator', scope: 'ent01' }],
  active_role: 'administrator',
  policy_seq: 7,
  demo: true,
  workplace: { id: 'WP-ADM' },
}
