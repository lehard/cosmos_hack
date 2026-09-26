<script setup lang="ts">
/**
 * Шапка (PRD §3a, Д-70, UI-4): спокойная — поиск изделия, уведомления,
 * целостность журнала «по данным сервера», токен, пользователь, справка, выход.
 * Момент просмотра появляется только при просмотре прошлого (с кнопкой
 * «Вернуться к текущему»); режим данных fixtures | live — в меню пользователя
 * одной строкой с пояснением. Имя системы — «Главный» (Д-65). Длинные имена и
 * роли — многоточие с подсказкой.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NButton, NIcon, NPopover, NTooltip } from 'naive-ui'
import { Help, Logout } from '@vicons/tabler'
import { useLogout, useSession } from '@/entities/session'
import { backendModeOf } from '@/shared/api'
import type { LiveStatus } from '@/shared/api/sse'
import { codeToKey } from '@/shared/i18n'
import { BrandMark } from '@/shared/ui'
import IntegrityIndicator from './header/IntegrityIndicator.vue'
import ItemSearch from './header/ItemSearch.vue'
import MomentIndicator from './header/MomentIndicator.vue'
import NotificationsBell from './header/NotificationsBell.vue'
import TokenStatus from './header/TokenStatus.vue'

defineProps<{ live: LiveStatus }>()

const { t, te } = useI18n()
const router = useRouter()
const session = useSession()
const logout = useLogout()

const s = computed(() => session.data.value?.data)
const mode = computed(() => backendModeOf(session.data.value))

/** Название роли — по словарю продукта, иначе как в политике. */
const roleTitle = computed(() => {
  const role = s.value?.role
  if (!role) return ''
  const key = `roles.${codeToKey(role.id)}`
  return te(key) ? t(key) : role.title
})

async function onLogout() {
  await logout.mutateAsync().catch(() => undefined)
  await router.push({ name: 'login' })
}
</script>

<template>
  <div class="header">
    <RouterLink to="/desk" class="brand" :title="t('shell.brand.tagline')">
      <BrandMark />
    </RouterLink>

    <ItemSearch class="search" />

    <div class="right">
      <MomentIndicator />
      <NTooltip v-if="live === 'closed'">
        <template #trigger>
          <span class="live-off" />
        </template>
        {{ t('shell.header.liveOff') }}
      </NTooltip>
      <IntegrityIndicator />
      <TokenStatus />
      <span class="divider" aria-hidden="true" />
      <NotificationsBell />
      <NPopover v-if="s" trigger="click" placement="bottom-end" :style="{ maxWidth: '360px' }">
        <template #trigger>
          <button type="button" class="user" data-testid="header-user" :title="[s.user.name, roleTitle].join(' · ')" :aria-label="t('shell.header.userMenu')">
            <span class="user-name ant-ellipsis">{{ s.user.name }}</span>
            <span class="user-role ant-ellipsis">
              {{ roleTitle }}<template v-if="s.shift"> · {{ t('common.words.shift') }}: {{ s.shift.title }}</template>
              <template v-if="s.workplace"> · {{ t('common.header.workplace', { workplace: s.workplace.title }) }}</template>
            </span>
          </button>
        </template>
        <div class="user-menu" data-testid="user-menu">
          <p class="menu-name ant-wrap">{{ s.user.name }}</p>
          <p class="menu-line ant-wrap">{{ roleTitle }}</p>
          <p v-if="s.shift" class="menu-line ant-wrap">{{ t('common.words.shift') }}: {{ s.shift.title }}</p>
          <p v-if="s.workplace" class="menu-line ant-wrap">{{ t('common.header.workplace', { workplace: s.workplace.title }) }}</p>
          <div v-if="mode" class="menu-mode" data-testid="backend-mode" :data-mode="mode">
            <span class="menu-caption ant-ellipsis">{{ t('shell.header.dataMode') }}</span>
            <p class="menu-line ant-wrap">{{ mode === 'fixtures' ? t('hints.fixturesMode') : `${t('common.modes.backendLive')} — ${t('shell.header.liveModeHint')}` }}</p>
          </div>
        </div>
      </NPopover>
      <NTooltip>
        <template #trigger>
          <NButton quaternary circle :aria-label="t('common.actions.helpForRole')" @click="router.push({ name: 'help' })">
            <template #icon><NIcon><Help /></NIcon></template>
          </NButton>
        </template>
        {{ t('common.actions.helpForRole') }}
      </NTooltip>
      <NTooltip>
        <template #trigger>
          <NButton quaternary circle :aria-label="t('common.actions.logout')" @click="onLogout">
            <template #icon><NIcon><Logout /></NIcon></template>
          </NButton>
        </template>
        {{ t('common.actions.logout') }}
      </NTooltip>
    </div>
  </div>
</template>

<style scoped>
.header {
  display: flex;
  gap: var(--ant-space-6);
  align-items: center;
  height: 100%;
  padding: 0 var(--ant-space-6);
}

.brand {
  display: inline-flex;
  flex: none;
  color: inherit;
  text-decoration: none;
}

.search {
  flex: 0 1 var(--ant-w-search);
  min-width: 180px;
}

.right {
  display: flex;
  flex: 1 1 auto;
  gap: var(--ant-space-4);
  align-items: center;
  justify-content: flex-end;
  min-width: 0;
  font-size: var(--ant-fs-sm);
}

.divider {
  flex: none;
  width: 1px;
  height: 24px;
  background: var(--ant-border);
}

.user {
  display: flex;
  flex: 0 1 auto;
  flex-direction: column;
  align-items: flex-start;
  min-width: 0;
  max-width: 280px;
  padding: 2px 6px;
  border: 0;
  border-radius: var(--ant-radius-sm);
  background: none;
  color: inherit;
  font: inherit;
  line-height: var(--ant-lh-tight);
  text-align: left;
  cursor: pointer;
}

.user > span {
  max-width: 100%;
}

.user:hover {
  background: var(--ant-surface-hover);
}

.user-menu {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-1);
  font-size: var(--ant-fs-sm);
}

.user-menu p {
  margin: 0;
}

.menu-name {
  font-weight: var(--ant-fw-bold);
}

.menu-line {
  color: var(--ant-text-2);
}

.menu-mode {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: var(--ant-space-2);
  padding-top: var(--ant-space-2);
  border-top: 1px solid var(--ant-border);
}

.menu-caption {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.user-name {
  font-weight: var(--ant-fw-bold);
}

.user-role {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-xs);
}

.live-off {
  display: inline-block;
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--ant-status-attention);
}
</style>
