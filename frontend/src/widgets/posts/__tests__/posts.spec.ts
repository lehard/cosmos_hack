// Панель «Посты» (FR-6): присутствие по СКУД и ключу, текущее изделие → паспорт,
// «неизвестно» ≠ «на месте», ошибка входа, пока операции нет.
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { workplaceKeys, type PostRow } from '@/entities/workplace'
import type { Envelope } from '@/shared/api/pending'
import { i18n } from '@/shared/i18n'
import PostsWidget from '../ui/PostsWidget.vue'

/** Образец постов сварочного цеха — только для тестов. */
const rows = (): PostRow[] => [
  { workplace_id: 'WP-W2', station: 'Сварочный пост 2', assigned: { person_id: 'P-17', display: 'Сварщик С-17' }, presence: 'present', current_item: { item_id: 'ENT:FL-0041', label: 'ФЛ-0041' } },
  { workplace_id: 'WP-W3', station: 'Сварочный пост 3', assigned: { person_id: 'P-21', display: 'Сварщик С-21' }, presence: 'key_missing', current_item: null },
  { workplace_id: 'WP-K3', station: 'ЗТ-3 ОТК', assigned: { person_id: 'P-05', display: 'Контролёр К-05' }, presence: 'owner_absent', current_item: null },
  { workplace_id: 'WP-A1', station: 'Сборка 1', assigned: null, presence: 'not_assigned', current_item: null },
]

async function mountWidget(seed: PostRow[] | null) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/desk', component: { template: '<div />' } },
      { path: '/items/:id', name: 'item', component: { template: '<div />' } },
    ],
  })
  await router.push('/desk')
  if (seed) queryClient.setQueryData<Envelope<PostRow[]>>(workplaceKeys.list('posts', {}, { axis: 'occurred' }), { data: seed, headers: new Headers({ 'Ant-Backend': 'live' }) })
  const w = mount(PostsWidget, {
    props: { widgetId: 'posts', titleKey: 'liveMap.posts.title', slotId: 'posts', slice: {}, density: 'comfortable' },
    global: { plugins: [pinia, i18n, router, [VueQueryPlugin, { queryClient }]] },
  })
  await flushPromises()
  return { w, router }
}

describe('виджет «Посты»', () => {
  it('участок — назначен — на месте ли — текущее изделие', async () => {
    const { w } = await mountWidget(rows())
    const text = (id: string) => w.find(`[data-workplace="${id}"]`).text()
    expect(text('WP-W2')).toContain('Сварщик С-17')
    expect(text('WP-W2')).toContain('На месте')
    expect(text('WP-W3')).toContain('По графику на месте — ключ не вставлен')
    expect(text('WP-K3')).toContain('Ключ вставлен — владельца нет в зоне')
    expect(text('WP-A1')).toContain('Никто не назначен')
    expect(w.find('.widget-frame').attributes('data-state')).toBe('normal')
    expect(w.find('.widget-frame').attributes('data-mode')).toBe('live')
  })

  it('текущее изделие → паспорт (FR-7)', async () => {
    const { w, router } = await mountWidget(rows())
    await w.find('[data-workplace="WP-W2"] .item').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/items/ENT:FL-0041')
  })

  it('нет данных присутствия — «оценка невозможна», а не «на месте» (NFR-UI-4)', async () => {
    const { w } = await mountWidget([{ ...rows()[0]!, presence: 'unknown' }, { ...rows()[1]!, presence: 'absent' }])
    expect(w.find('.widget-frame').attributes('data-state')).toBe('unable_to_assess')
    expect(w.find('[data-presence="unknown"]').text()).toContain('Нет данных — неизвестно')
    expect(w.find('[data-presence="absent"]').text()).toContain('Нет на месте')
  })

  it('операции ещё нет — «ошибка входа»', async () => {
    const { w } = await mountWidget(null)
    await vi.waitFor(() => expect(w.find('.widget-frame').attributes('data-state')).toBe('input_error'))
  })
})
