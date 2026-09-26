<script setup lang="ts">
/**
 * Шапка (PRD §3a): пользователь, роль, смена, статус токена, индикатор
 * целостности «по данным сервера», уведомления, поиск по номеру детали; плюс
 * момент просмотра, режим данных fixtures | live, справка и выход.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NButton, NIcon, NSpace, NTag, NText, NTooltip } from 'naive-ui'
import { Help, Logout } from '@vicons/tabler'
import { useLogout, useSession } from '@/entities/session'
import { backendModeOf } from '@/shared/api'
import type { LiveStatus } from '@/shared/api/sse'
import { codeToKey } from '@/shared/i18n'
import { statusPalette } from '@/shared/api/generated/statuses'
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
    <RouterLink to="/desk" class="brand">
      <NText strong>{{ t('common.appName') }}</NText>
    </RouterLink>

    <ItemSearch class="search" />

    <NSpace align="center" :size="16" :wrap="false" class="right">
      <MomentIndicator />
      <NTag v-if="mode" size="small" data-testid="backend-mode">
        {{ t(mode === 'fixtures' ? 'common.modes.backendFixtures' : 'common.modes.backendLive') }}
      </NTag>
      <NTooltip v-if="live === 'closed'">
        <template #trigger>
          <span class="live-off" :style="{ background: statusPalette.attention }" />
        </template>
        {{ t('shell.header.liveOff') }}
      </NTooltip>
      <IntegrityIndicator />
      <TokenStatus />
      <NotificationsBell />
      <div v-if="s" class="user" data-testid="header-user">
        <NText strong>{{ s.user.name }}</NText>
        <NText depth="3">
          {{ roleTitle }}<template v-if="s.shift"> · {{ t('common.words.shift') }}: {{ s.shift.title }}</template>
          <template v-if="s.workplace"> · {{ t('common.header.workplace', { workplace: s.workplace.title }) }}</template>
        </NText>
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
    </NSpace>
  </div>
</template>

<style scoped>
.header {
  display: flex;
  gap: 24px;
  align-items: center;
  padding: 8px 24px;
  min-height: 40px;
}

.brand {
  color: inherit;
  text-decoration: none;
  white-space: nowrap;
}

.search {
  width: 320px;
}

.right {
  margin-left: auto;
}

.user {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}

.live-off {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
</style>
