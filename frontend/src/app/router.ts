/**
 * Маршруты интерфейса (AD-21, FR-128). Вход → стол своей роли; стол — данные
 * с сервера, поэтому маршрут один на все роли: /desk/‹вкладка›.
 * Охранник: без сеанса — на экран входа (с возвратом на запрошенную страницу);
 * с сеансом — сообщает расширению подписи, кто вошёл.
 */
import { createRouter, createWebHistory } from 'vue-router'
import { sessionQueryOptions } from '@/entities/session'
import { setTokenPerson } from '@/shared/lib/token-agent'
import ShellLayout from './layout/ShellLayout.vue'
import { queryClient } from './providers/query'

declare module 'vue-router' {
  interface RouteMeta {
    /** Страница доступна без сеанса. */
    public?: boolean
  }
}

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/pages/login'), meta: { public: true } },
    {
      path: '/',
      component: ShellLayout,
      children: [
        { path: '', redirect: { name: 'desk' } },
        { path: 'desk/:tab?', name: 'desk', component: () => import('@/pages/desk') },
        { path: 'items/:id', name: 'item', component: () => import('@/pages/item') },
        // Эпик 11: карточка несоответствия — сюда ведут ссылки features/drill-down (FR-7).
        { path: 'nonconformities/:id', name: 'nonconformity', component: () => import('@/pages/nonconformity') },
        { path: 'help/:role?', name: 'help', component: () => import('@/pages/help') },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  try {
    const session = await queryClient.fetchQuery(sessionQueryOptions())
    // Расширение подписи выбирает ключ вошедшего (ключи многих персон, Д-72).
    setTokenPerson({ id: session.data.user.id, name: session.data.user.name })
    return true
  } catch {
    // Нет сеанса (401), операция ещё не реализована (501) или нет связи —
    // экран входа сам покажет причину.
    return { name: 'login', query: to.fullPath !== '/' ? { next: to.fullPath } : {} }
  }
})
