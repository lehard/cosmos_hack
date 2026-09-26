/**
 * Точка входа интерфейса ant (слой app, FSD): роутер, Pinia (состояние интерфейса
 * и сеанса), Vue Query (серверное состояние, AD-21), тексты vue-i18n (NFR-UI-3),
 * права @casl/vue (AD-15). Шрифты — локальные пакеты (NFR-UI-2).
 */
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import '@fontsource/pt-sans/400.css'
import '@fontsource/pt-sans/700.css'
import '@fontsource/pt-mono/400.css'
import { i18n } from '@/shared/i18n'
import './styles.css'
import App from './App.vue'
import { installAccess } from './providers/access'
import { queryClient } from './providers/query'
import { router } from './router'

const app = createApp(App).use(createPinia()).use(i18n).use(VueQueryPlugin, { queryClient })
installAccess(app)
app.use(router).mount('#app')
