<script setup lang="ts">
/**
 * Список внешних систем (FR-157): система, состояние «включена / выключена /
 * стенд» (неустановленная — «не установлена»), живой канал, последний обмен,
 * очередь и карантин исходящих. Строка — кнопка: открывает окно записи.
 */
import { useI18n } from 'vue-i18n'
import type { IntegrationEntry, IntegrationList } from '@/entities/integration'
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import { DataTable, EmptyState } from '@/shared/ui'

defineProps<{ list: IntegrationList }>()
const emit = defineEmits<{ open: [system: string] }>()
const { t, te, d } = useI18n()

const STATE_TONE: Record<string, StatusTone> = { enabled: 'success', stand: 'info', disabled: 'neutral' }
const CHANNEL_TONE: Record<string, StatusTone> = { ok: 'success', degraded: 'attention', disabled: 'neutral' }

/** Название системы для людей; неизвестная — её код. */
const systemName = (s: string) => (te(`widgets.integrations.systems.${s}`) ? t(`widgets.integrations.systems.${s}`) : s)
const stateText = (e: IntegrationEntry) => (e.installed ? t(`widgets.integrations.states.${e.state}`) : t('widgets.integrations.states.notInstalled'))
const stateColor = (e: IntegrationEntry) => statusPalette[e.installed ? (STATE_TONE[e.state] ?? 'neutral') : 'neutral']
const channelColor = (c: string) => statusPalette[CHANNEL_TONE[c] ?? 'neutral']
const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : '—')
</script>

<template>
  <div class="integrations ant-box" data-testid="integrations">
    <p class="hint ant-wrap">{{ t('widgets.integrations.hint') }}</p>
    <p v-if="list.profile === 'prod'" class="hint ant-wrap" data-testid="prod-no-stand">{{ t('widgets.integrations.prodNoStand') }}</p>
    <EmptyState v-if="!list.items.length" compact :title="t('widgets.integrations.empty')" />
    <DataTable v-else>
      <thead>
        <tr>
          <th>{{ t('widgets.integrations.system') }}</th>
          <th>{{ t('widgets.integrations.state') }}</th>
          <th>{{ t('widgets.integrations.channel') }}</th>
          <th>{{ t('widgets.integrations.lastExchange') }}</th>
          <th>{{ t('widgets.integrations.queue') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="e in list.items"
          :key="e.system"
          class="row"
          :class="{ absent: !e.installed }"
          :data-system="e.system"
          :data-state="e.installed ? e.state : 'not_installed'"
          tabindex="0"
          @click="emit('open', e.system)"
          @keydown.enter="emit('open', e.system)"
        >
          <td>
            <button type="button" class="link ant-wrap" data-testid="integration-open" @click.stop="emit('open', e.system)">{{ systemName(e.system) }}</button>
          </td>
          <td>
            <span class="tag ant-box">
              <span class="dot" :style="{ background: stateColor(e) }" aria-hidden="true" />
              <span class="ant-ellipsis">{{ stateText(e) }}</span>
            </span>
            <div v-if="e.installed && e.default" class="muted ant-ellipsis">{{ t('widgets.integrations.byDefault') }}</div>
          </td>
          <td>
            <span v-if="e.channel" class="tag ant-box">
              <span class="dot" :style="{ background: channelColor(e.channel) }" aria-hidden="true" />
              <span class="ant-ellipsis">{{ t(`widgets.integrations.channels.${e.channel}`) }}</span>
            </span>
            <span v-else class="muted">—</span>
          </td>
          <td><span class="ant-ellipsis">{{ time(e.last_exchange_at) }}</span></td>
          <td>
            <span v-if="e.queued != null || e.quarantined != null" :class="{ warn: (e.quarantined ?? 0) > 0 }">{{ e.queued ?? 0 }} / {{ e.quarantined ?? 0 }}</span>
            <span v-else class="muted">—</span>
          </td>
        </tr>
      </tbody>
    </DataTable>
  </div>
</template>

<style scoped>
.integrations {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

.hint {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.row {
  cursor: pointer;
}

.row.absent {
  color: var(--ant-text-3);
}

.tag {
  display: inline-flex;
  gap: var(--ant-space-2);
  align-items: center;
  max-width: 100%;
}

.dot {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.link:hover,
.link:focus-visible {
  text-decoration: underline;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.warn {
  color: var(--ant-status-danger);
}
</style>
