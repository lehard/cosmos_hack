<script setup lang="ts">
/**
 * Источники и интеграции — представление (FR-30, FR-33; AD-26; кейс О3/Т7):
 * устройства и их ключи (акт ввода), последнее сообщение и номер, пропуски
 * в номерах, карантин, расхождение часов; отключение и включение источника с
 * основанием. Ниже — статус обмена с внешними системами (1С, MES, КОМПАС).
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NInput } from 'naive-ui'
import type { ErpChannel } from '@/entities/erp-message'
import type { SourceView } from '@/entities/quarantine'
import { statusPalette, type StatusTone } from '@/shared/api/generated/statuses'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, DataTable } from '@/shared/ui'

withDefaults(
  defineProps<{
    sources: SourceView[]
    channels?: ErpChannel[] | null
    channelsError?: unknown
    canAct?: boolean
    busy?: boolean
    error?: unknown
    /** Источник, по которому команда принята. */
    done?: string | null
    density?: Density
  }>(),
  { channels: null, channelsError: undefined, canAct: true, busy: false, error: undefined, done: null, density: 'comfortable' },
)
const emit = defineEmits<{ switch: [source: SourceView, to: 'disable' | 'enable', reason: string] }>()
const { t, te, d } = useI18n()
const problemText = useProblemText()

const TONE: Record<string, StatusTone> = { active: 'success', ok: 'success', disabled: 'neutral', loss_suspected: 'attention', degraded: 'attention', unknown_key: 'danger' }
const color = (code: string) => statusPalette[TONE[code] ?? 'neutral']
const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : t('widgets.admin.sources.never'))
const systemText = (s: string) => (te(`widgets.admin.system.${s}`) ? t(`widgets.admin.system.${s}`) : s)

// Основание вводится в строке источника, для которого нажата команда.
const editing = ref<{ id: string; to: 'disable' | 'enable' } | null>(null)
const reason = ref('')
function ask(s: SourceView): void {
  editing.value = { id: s.source_id, to: s.state === 'disabled' ? 'enable' : 'disable' }
  reason.value = ''
}
function confirm(s: SourceView): void {
  if (!editing.value || !reason.value.trim()) return
  emit('switch', s, editing.value.to, reason.value.trim())
  editing.value = null
}
</script>

<template>
  <div class="sources" :class="`density-${density}`" data-testid="sources">
    <section>
      <h4>{{ t('admin.devices') }}</h4>
      <p v-if="!sources.length" class="muted">{{ t('widgets.admin.sources.noSources') }}</p>
      <DataTable v-else>
        <thead>
          <tr>
            <th>{{ t('widgets.admin.sources.source') }}</th>
            <th>{{ t('widgets.admin.sources.state') }}</th>
            <th>{{ t('widgets.admin.sources.key') }}</th>
            <th>{{ t('widgets.admin.sources.lastReceived') }}</th>
            <th>{{ t('widgets.admin.sources.lastSeq') }}</th>
            <th>{{ t('widgets.admin.sources.gaps') }}</th>
            <th>{{ t('widgets.admin.sources.quarantined') }}</th>
            <th>{{ t('widgets.admin.sources.clockSkew') }}</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in sources" :key="s.source_id" :data-source="s.source_id" :data-state="s.state">
            <td>
              <strong>{{ s.source_id }}</strong>
              <div class="muted">{{ s.source_kind }}</div>
            </td>
            <td><span class="dot" :style="{ background: color(s.state) }" aria-hidden="true" />{{ t(`widgets.admin.sources.sourceState.${codeToKey(s.state)}`) }}</td>
            <td>{{ s.key_ref ?? '—' }}</td>
            <td>{{ time(s.last_received_at) }}</td>
            <td>{{ s.last_seq ?? '—' }}</td>
            <td :class="{ warn: s.gap_count > 0 }">{{ s.gap_count }}</td>
            <td>{{ s.quarantined }}</td>
            <td>{{ s.clock_skew_ms ?? '—' }}</td>
            <td class="act">
              <span v-if="done === s.source_id" class="ok">{{ t('widgets.admin.sources.switched') }}</span>
              <form v-else-if="editing?.id === s.source_id" class="reason" @submit.prevent="confirm(s)">
                <NInput v-model:value="reason" size="tiny" :placeholder="t('widgets.admin.sources.reason')" :aria-label="t('widgets.admin.sources.reason')" />
                <ActionButton size="tiny" attr-type="submit" type="primary" :disabled="!canAct || busy || !reason.trim()" data-testid="confirm" :label="t(editing.to === 'enable' ? 'widgets.admin.sources.enable' : 'widgets.admin.sources.disable')" />
                <ActionButton size="tiny" quaternary @click="editing = null" :label="t('common.actions.cancel')" />
              </form>
              <ActionButton v-else size="tiny" :disabled="!canAct || busy" data-testid="switch" @click="ask(s)" :label="t(s.state === 'disabled' ? 'widgets.admin.sources.enable' : 'widgets.admin.sources.disable')" />
            </td>
          </tr>
        </tbody>
      </DataTable>
      <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
    </section>

    <section data-testid="channels">
      <h4>{{ t('admin.integrationsStatus') }}</h4>
      <p v-if="channelsError" class="error">{{ problemText(channelsError) }}</p>
      <ul v-else-if="channels" class="plain">
        <li v-for="c in channels" :key="c.system" :data-channel="c.system">
          <span class="dot" :style="{ background: color(c.state) }" aria-hidden="true" />
          <strong>{{ systemText(c.system) }}</strong>
          {{ t(`widgets.admin.sources.channelState.${c.state}`) }}
          <span class="muted">{{ c.endpoint }}<template v-if="c.stand"> · {{ t('widgets.admin.sources.stand') }}</template></span>
          <span class="muted">{{ t('admin.lastExchange', { time: time(c.last_exchange_at) }) }}</span>
          <span class="muted">{{ t('widgets.admin.sources.queued') }}: {{ c.queued }} · {{ t('widgets.admin.sources.quarantined') }}: {{ c.quarantined }}</span>
          <span v-if="c.detail" class="muted">{{ c.detail }}</span>
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.sources {
  display: flex;
  flex-direction: column;
  gap: 12px;
  font-size: var(--ant-fs-body);
}

h4 {
  margin: 0 0 4px;
  font-size: 1em;
}

.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  margin-right: 6px;
  border-radius: 50%;
}

.reason {
  display: flex;
  gap: 4px;
}

.plain {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.plain li {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 8px;
  align-items: center;
}

.warn,
.error {
  color: var(--ant-status-danger);
}

.ok {
  color: var(--ant-status-success);
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
