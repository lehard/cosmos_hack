<script setup lang="ts">
/**
 * Левое меню стола (Д-73): разделы стола роли вместо вкладок сверху. Видно на
 * любой странице оболочки (паспорт, справка) — вернуться к разделу в один
 * щелчок. Раздел один — меню нет. Развёрнуто по умолчанию; свёрнутое состояние
 * помнит браузер. Alt+1…9 — раздел по номеру.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  Affiliate, Alarm, ChartLine, ClipboardCheck, Clock, Components, Dashboard, Database, Eye, FileAnalytics, FileText,
  Gavel, History, Key, Layout2, ListDetails, ListCheck, Lock, PlugConnected, Point, Route, Search, Shield, ShieldCheck,
  Signature, Sitemap, Target, Terminal,
  Tool, Users,
} from '@vicons/tabler'
import { useDesk } from '@/entities/desk'
import { SideNav, type SideNavItem } from '@/shared/ui'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const desk = useDesk()

/**
 * Значок раздела по его id в normative/desks (оформление, не данные стола);
 * новый раздел без строки здесь получает общий значок.
 */
const ICONS: Record<string, Component> = {
  overview: Dashboard,
  processes: Route,
  posts: Tool,
  proposals: ClipboardCheck,
  analytics: ChartLine,
  'data-deficit': Search,
  process: Sitemap,
  station: Layout2,
  people: Users,
  tasks: ListCheck,
  shift: Clock,
  causes: FileAnalytics,
  circumstances: History,
  risk: Target,
  investigation: Target,
  workbench: Dashboard,
  actions: ClipboardCheck,
  journal: ListDetails,
  'decisions-journal': Gavel,
  integrity: Shield,
  'critical-actions': Alarm,
  'security-events': Lock,
  grants: Key,
  scenarios: Terminal,
  access: Key,
  sources: Database,
  integrations: PlugConnected,
  'vision-adaptation': Eye,
  components: Components,
  queue: ListCheck,
  decision: Signature,
  terminal: Terminal,
  documents: FileText,
  quality: ShieldCheck,
  federation: Affiliate,
}

/**
 * Меню «Работа / Справочно»: справочные разделы (документы, аналитика; у
 * руководителя — ещё процессы) — в конце под тихой подписью, чтобы меню не выглядело
 * девятью одинаково важными направлениями. Порядок внутри групп — как на столе.
 */
const REFERENCE = new Set(['documents', 'analytics'])
// Для руководителя процессы — справочно (управляет через центр и карту); технологу «Техпроцессы» — часть рабочего цикла.
const REFERENCE_BY_ROLE: Record<string, readonly string[]> = { production_manager: ['process'] }
const isReference = (id: string) => REFERENCE.has(id) || (REFERENCE_BY_ROLE[desk.data.value?.data?.role ?? ''] ?? []).includes(id)
const tabs = computed(() => {
  const all = desk.data.value?.data?.tabs ?? []
  return [...all.filter((x) => !isReference(x.id)), ...all.filter((x) => isReference(x.id))]
})
const items = computed<SideNavItem[]>(() => {
  const firstRef = tabs.value.findIndex((x) => isReference(x.id))
  // Группа «Справочно» — только если до неё есть рабочие разделы.
  return tabs.value.map((tab, i) => ({
    id: tab.id,
    label: t(tab.title_key),
    icon: ICONS[tab.id] ?? Point,
    hint: i < 9 ? `Alt+${i + 1}` : undefined,
    groupStart: i === firstRef && firstRef > 0 ? t('shell.nav.reference') : undefined,
  }))
})

/** Выбранный раздел — только на странице стола. */
const active = computed(() => {
  if (route.name !== 'desk') return undefined
  const wanted = typeof route.params.tab === 'string' ? route.params.tab : ''
  return (tabs.value.find((tab) => tab.id === wanted) ?? tabs.value[0])?.id
})

const select = (id: string) => void router.push({ name: 'desk', params: { tab: id } })

const STORAGE_KEY = 'ant.nav.collapsed'
const collapsed = ref(readCollapsed())
function readCollapsed(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    return false
  }
}
watch(collapsed, (v) => {
  try {
    localStorage.setItem(STORAGE_KEY, v ? '1' : '0')
  } catch {
    // Хранилище недоступно — состояние живёт до перезагрузки.
  }
})

/** Alt+1…9 — раздел по номеру (e.code: цифра верхнего ряда при любой раскладке). */
function onKey(e: KeyboardEvent): void {
  if (!e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return
  const m = /^Digit([1-9])$/.exec(e.code)
  const tab = m ? tabs.value[Number(m[1]) - 1] : undefined
  if (!tab || tabs.value.length < 2) return
  e.preventDefault()
  if (tab.id !== active.value) select(tab.id)
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <SideNav
    v-if="items.length > 1"
    v-model:collapsed="collapsed"
    :items="items"
    :active="active"
    :label="t('shell.nav.label')"
    :collapse-label="t('shell.nav.collapse')"
    :expand-label="t('shell.nav.expand')"
    data-testid="desk-nav"
    @select="select"
  />
</template>
