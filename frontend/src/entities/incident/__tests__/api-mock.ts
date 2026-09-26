// Подмена сети для тестов контейнеров: сгенерированный клиент зовёт глобальный
// fetch — тест отвечает по «МЕТОД путь» заготовленным телом и запоминает запросы.
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { vi } from 'vitest'
import type { Component } from 'vue'
import { i18n } from '@/shared/i18n'

export interface Call {
  method: string
  path: string
  body: unknown
}

/**
 * Ответы API по ключу `GET /api/v1/…` (путь без параметров запроса).
 * Нет ключа — 404 problem+json. Ответы помечены `Ant-Backend: fixtures`.
 */
export function mockApi(routes: Record<string, unknown>): Call[] {
  const calls: Call[] = []
  vi.stubGlobal('fetch', async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    const path = decodeURIComponent(new URL(url, 'http://ant.local').pathname)
    calls.push({ method, path, body: init?.body ? JSON.parse(String(init.body)) : undefined })
    const key = `${method} ${path}`
    const headers = { 'Content-Type': 'application/json', 'Ant-Backend': 'fixtures' }
    if (!(key in routes)) {
      return new Response(JSON.stringify({ type: 'urn:ant:problem:reference.not_found', title: 'Не найдено', status: 404, code: 'reference.not_found' }), { status: 404, headers })
    }
    return new Response(JSON.stringify(routes[key]), { status: 200, headers })
  })
  return calls
}

/** Смонтировать контейнер виджета с Pinia, текстами и Vue Query. */
export async function mountWidget(component: Component, props: Record<string, unknown>) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const w = mount(component, {
    props: { slotId: 's', slice: {}, density: 'compact', ...props },
    global: { plugins: [createPinia(), i18n, [VueQueryPlugin, { queryClient }]] },
  })
  await settle()
  return w
}

/** Дождаться ответов и перерисовки. */
export async function settle(): Promise<void> {
  for (let i = 0; i < 6; i++) {
    await flushPromises()
    await new Promise((r) => setTimeout(r, 0))
  }
}

/** Права пользователя (`access.permission.list`). */
export const permissions = (actions: [string, string][], policy_seq = 7) => ({
  policy_seq,
  items: actions.map(([action, subject]) => ({ action, subject, action_class: 'record' })),
})
