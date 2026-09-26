<script setup lang="ts">
/**
 * Карантин сообщений — представление (FR-41, AD-26; кейс §5.1 S12): записи с
 * кодом причины, источником и временем; выбранная — с исходным содержимым,
 * отпечатком и адресом; переобработка (принимается один раз, AD-7) или отказ
 * с основанием. Номер изделия без человека не подставляется.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NCheckbox, NInput, NRadioButton, NRadioGroup } from 'naive-ui'
import type { QuarantineEntry } from '@/entities/quarantine'
import { errorCatalog } from '@/shared/api/generated/errors'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton } from '@/shared/ui'

const props = withDefaults(
  defineProps<{
    entries: QuarantineEntry[]
    filter: 'open' | 'all'
    selected?: string | null
    /** Выбранная запись целиком (с содержимым, если есть права). */
    detail?: QuarantineEntry | null
    detailError?: unknown
    /** Сводка на чужом столе: без содержимого и команд. */
    summary?: boolean
    canAct?: boolean
    busy?: boolean
    error?: unknown
    /** Запись, переобработка которой принята. */
    done?: string | null
    density?: Density
  }>(),
  { selected: null, detail: null, detailError: undefined, summary: false, canAct: true, busy: false, error: undefined, done: null, density: 'comfortable' },
)
const emit = defineEmits<{
  'update:filter': [filter: 'open' | 'all']
  select: [id: string]
  reprocess: [entry: QuarantineEntry, reason: string, discard: boolean]
}>()
const { t, d } = useI18n()
const problemText = useProblemText()

const time = (iso: string) => d(new Date(iso), 'dateTime')
/** Заголовок кода причины из каталога ошибок (contracts/errors.yaml). */
const codeTitle = (code: string) => (errorCatalog as Record<string, { title: string } | undefined>)[code]?.title ?? code

const current = computed(() => props.detail ?? props.entries.find((e) => e.quarantine_id === props.selected) ?? null)
const reason = ref('')
const discard = ref(false)
watch(
  () => props.selected,
  () => {
    reason.value = ''
    discard.value = false
  },
)
function submit(): void {
  if (current.value && reason.value.trim()) emit('reprocess', current.value, reason.value.trim(), discard.value)
}
</script>

<template>
  <div class="quarantine" :class="`density-${density}`" data-testid="quarantine">
    <NRadioGroup v-if="!summary" :value="filter" size="small" @update:value="(v: 'open' | 'all') => emit('update:filter', v)">
      <NRadioButton value="open" data-testid="filter-open">{{ t('widgets.admin.quarantine.filter.open') }}</NRadioButton>
      <NRadioButton value="all" data-testid="filter-all">{{ t('widgets.admin.quarantine.filter.all') }}</NRadioButton>
    </NRadioGroup>
    <p v-if="!entries.length" class="muted" data-testid="empty">{{ t('empty.quarantineEmpty') }}</p>
    <ol class="rows">
      <li
        v-for="e in entries"
        :key="e.quarantine_id"
        class="row"
        tabindex="0"
        :data-id="e.quarantine_id"
        :data-state="e.state"
        :aria-selected="e.quarantine_id === selected"
        @click="emit('select', e.quarantine_id)"
        @keydown.enter.prevent="emit('select', e.quarantine_id)"
      >
        <div class="line">
          <strong class="code" :title="e.problem_code">{{ codeTitle(e.problem_code) }}</strong>
          <span class="muted">{{ t(`widgets.admin.quarantine.entryState.${codeToKey(e.state)}`) }}</span>
        </div>
        <div class="line muted">
          <span>{{ t('admin.quarantine.source') }}: {{ e.source_id }}<template v-if="e.source_seq !== undefined"> № {{ e.source_seq }}</template></span>
          <span>{{ t('admin.quarantine.receivedAt') }}: {{ time(e.quarantined_at) }}</span>
          <span v-if="e.event_type">{{ e.event_type }}</span>
        </div>
        <div v-if="e.detail" class="line detail">{{ e.detail }}</div>
      </li>
    </ol>

    <section v-if="!summary && current" class="card" data-testid="detail">
      <h4>{{ codeTitle(current.problem_code) }} <code class="muted">{{ current.problem_code }}</code></h4>
      <dl class="facts">
        <dt>{{ t('admin.quarantine.source') }}</dt>
        <dd>{{ current.source_id }}</dd>
        <template v-if="current.event_type">
          <dt>{{ t('widgets.admin.quarantine.eventType') }}</dt>
          <dd>{{ current.event_type }}</dd>
        </template>
        <dt>{{ t('widgets.admin.quarantine.fingerprint') }}</dt>
        <dd><code>{{ current.fingerprint }}</code></dd>
        <dt>{{ t('widgets.admin.quarantine.material') }}</dt>
        <dd><code>{{ current.material_address }}</code></dd>
      </dl>
      <p v-if="detailError" class="error">{{ problemText(detailError) }}</p>
      <template v-else>
        <h5>{{ t('widgets.admin.quarantine.content') }}</h5>
        <pre v-if="current.content" data-testid="content">{{ current.content }}</pre>
        <p v-else class="muted">{{ t('widgets.admin.quarantine.noContent') }}</p>
      </template>
      <p class="muted">{{ t('admin.quarantine.noAutoBinding') }}</p>
      <p v-if="done === current.quarantine_id" class="ok" data-testid="done">{{ t('admin.quarantine.reprocessedOnce') }}</p>
      <form v-else-if="current.state === 'open' || current.state === 'still_invalid'" class="reprocess" @submit.prevent="submit">
        <NInput v-model:value="reason" size="small" :placeholder="t('widgets.admin.quarantine.reason')" :aria-label="t('widgets.admin.quarantine.reason')" data-testid="reason" />
        <NCheckbox v-model:checked="discard" data-testid="discard">{{ t('widgets.admin.quarantine.discard') }}</NCheckbox>
        <ActionButton overflow="wrap" type="primary" size="small" attr-type="submit" :disabled="!canAct || busy || !reason.trim()" data-testid="reprocess" :label="t('admin.quarantine.reprocess')" />
      </form>
      <NAlert v-if="error" type="error" :bordered="false" :show-icon="false" data-testid="command-error">{{ problemText(error) }}</NAlert>
    </section>
    <p v-else-if="!summary" class="muted">{{ t('widgets.admin.quarantine.select') }}</p>
  </div>
</template>

<style scoped>
.quarantine {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: var(--ant-fs-body);
}

.rows {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.row {
  padding: 6px 8px;
  border: 1px solid var(--ant-border);
  border-left: 3px solid var(--ant-status-attention);
  border-radius: var(--ant-radius-md);
  cursor: pointer;
}

.row[data-state='accepted'],
.row[data-state='discarded'] {
  border-left-color: var(--ant-status-muted);
}

.row[aria-selected='true'] {
  border-color: var(--ant-accent);
  background: var(--ant-accent-soft);
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: baseline;
}

.card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
}

h4,
h5 {
  margin: 0;
}

.facts {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 2px 12px;
  margin: 0;
}

.facts dt {
  color: var(--ant-text-3);
}

.facts dd {
  margin: 0;
  word-break: break-all;
}

pre {
  max-height: 200px;
  margin: 0;
  padding: 6px;
  overflow: auto;
  background: var(--ant-n-50);
  font-size: var(--ant-fs-meta);
  white-space: pre-wrap;
  word-break: break-all;
}

.reprocess {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.error {
  color: var(--ant-status-danger);
}

.ok {
  color: var(--ant-status-success);
}
</style>
