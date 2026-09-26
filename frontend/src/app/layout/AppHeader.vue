<script setup lang="ts">
/**
 * Шапка (PRD §3a): пользователь, роль, смена, статус токена, индикатор
 * целостности «по данным сервера», уведомления, поиск по номеру детали; плюс
 * момент просмотра, режим данных fixtures | live, справка и выход.
 * Имя системы — «Главный» (Д-65). Длинные имена и роли — многоточие с подсказкой.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NButton, NIcon, NTooltip } from 'naive-ui'
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
      <span v-if="mode" class="mode" data-testid="backend-mode">
        {{ t(mode === 'fixtures' ? 'common.modes.backendFixtures' : 'common.modes.backendLive') }}
      </span>
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
      <div v-if="s" class="user" data-testid="header-user" :title="[s.user.name, roleTitle].join(' · ')">
        <span class="user-name ant-ellipsis">{{ s.user.name }}</span>
        <span class="user-role ant-ellipsis">
          {{ roleTitle }}<template v-if="s.shift"> · {{ t('common.words.shift') }}: {{ s.shift.title }}</template>
          <template v-if="s.workplace"> · {{ t('common.header.workplace', { workplace: s.workplace.title }) }}</template>
        </span>
      </div>
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

.mode {
  flex: none;
  padding: 0 6px;
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-sm);
  color: var(--ant-text-2);
  font-size: var(--ant-fs-xs);
  line-height: 20px;
  white-space: nowrap;
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
  min-width: 0;
  max-width: 280px;
  line-height: var(--ant-lh-tight);
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
