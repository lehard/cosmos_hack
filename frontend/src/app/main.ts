// Точка входа интерфейса ant (слой app, FSD).
// Подключает роутер, Pinia (клиентское состояние) и Vue Query — единственное
// место серверного состояния (AD-21). Шрифты — локальные пакеты (NFR-UI-2).
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import '@fontsource/pt-sans/400.css'
import '@fontsource/pt-sans/700.css'
import '@fontsource/pt-mono/400.css'
import './styles.css'
import App from './App.vue'
import { router } from './router'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1, refetchOnWindowFocus: false },
  },
})

createApp(App).use(createPinia()).use(router).use(VueQueryPlugin, { queryClient }).mount('#app')
