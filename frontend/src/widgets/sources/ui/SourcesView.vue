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
      <ul v-else class="lines">
        <li v-for="s in sources" :key="s.source_id" class="line" :data-source="s.source_id" :data-state="s.state">
          <span class="name">
            <span class="dot" :style="{ background: color(s.state) }" :title="t(`widgets.admin.sources.sourceState.${codeToKey(s.state)}`)" />
            <strong>{{ s.source_id }}</strong>
            <span class="muted">{{ s.source_kind }}</span>
          </span>
          <span class="facts">
            <span>{{ t(`widgets.admin.sources.sourceState.${codeToKey(s.state)}`) }}</span>
            <span class="muted">{{ time(s.last_received_at) }}<template v-if="s.last_seq != null"> · № {{ s.last_seq }}</template></span>
            <span v-if="s.gap_count > 0" class="warn">{{ t('widgets.admin.sources.gaps') }}: {{ s.gap_count }}</span>
            <span v-if="s.quarantined" class="warn">{{ t('widgets.admin.sources.quarantined') }}: {{ s.quarantined }}</span>
            <span v-if="s.clock_skew_ms" class="muted">{{ t('widgets.admin.sources.clockSkew') }}: {{ s.clock_skew_ms }}</span>
            <span v-if="s.key_ref" class="muted" :title="t('widgets.admin.sources.key')">{{ s.key_ref }}</span>
          </span>
          <span class="act">
            <span v-if="done === s.source_id" class="ok">{{ t('widgets.admin.sources.switched') }}</span>
            <form v-else-if="editing?.id === s.source_id" class="reason" @submit.prevent="confirm(s)">
              <NInput v-model:value="reason" size="tiny" :placeholder="t('widgets.admin.sources.reason')" :aria-label="t('widgets.admin.sources.reason')" />
              <ActionButton size="tiny" attr-type="submit" type="primary" :disabled="!canAct || busy || !reason.trim()" data-testid="confirm" :label="t(editing.to === 'enable' ? 'widgets.admin.sources.enable' : 'widgets.admin.sources.disable')" />
              <ActionButton size="tiny" quaternary @click="editing = null" :label="t('common.actions.cancel')" />
            </form>
            <button v-else type="button" class="link" :disabled="!canAct || busy" data-testid="switch" @click="ask(s)">{{ t(s.state === 'disabled' ? 'widgets.admin.sources.enable' : 'widgets.admin.sources.disable') }}</button>
          </span>
        </li>
      </ul>
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
.lines {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.line {
  display: grid;
  grid-template-columns: minmax(200px, 1fr) 2fr auto;
  gap: var(--ant-space-1) var(--ant-space-4);
  align-items: baseline;
  padding: var(--ant-space-3) 0;
  border-bottom: 1px solid var(--ant-border);
}

.line:last-child {
  border-bottom: 0;
}

.name,
.facts {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-3);
  align-items: baseline;
  min-width: 0;
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent);
  font: inherit;
  cursor: pointer;
}

.link:disabled {
  opacity: 0.5;
  cursor: default;
}

.sources {
  display: flex;
  flex-direction: column;
  gap: calc(var(--ant-space-4) * 2);
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
