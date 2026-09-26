<script setup lang="ts">
/**
 * Общий журнал (PRD §3a; AD-2, AD-44, AD-46; FR-68, FR-106, FR-140; кейс §7.2,
 * Т3) — разделы «Журнал операций» и «Журнал решений» стола Аудитора ИБ.
 *
 * Вверху — полоса целостности: вердикт верификатора, когда проверено,
 * отпечаток отчёта, голова журнала. Ниже — отбор (вид записи, изделие, тип
 * события или семейство) и таблица записей: исходные события, выводы системы,
 * решения людей и служебные записи различимы; у каждой — источник и статус
 * подписи. «Показать ещё» дочитывает страницы по курсору. Щелчок по строке —
 * правое окно записи (Д-70) со всеми полями, связями и содержимым.
 *
 * Срез стола: `entry_kind` — вид записи по умолчанию (fact | reaction |
 * decision | service); нет — все записи.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NRadioButton, NRadioGroup, NSelect } from 'naive-ui'
import {
  asEntryKind,
  effectiveIntegrity,
  JOURNAL_ENTRY_KINDS,
  useIntegrity,
  useJournalEntries,
  useJournalEntry,
  useJournalHead,
  useVerifierReports,
  type JournalEntryKind,
  type JournalFilter,
} from '@/entities/integrity'
import { useDrillDown } from '@/features/drill-down'
import type { WidgetProps } from '@/shared/config/widget'
import { naiveSizeOf } from '@/shared/config/widget'
import { useServerNow } from '@/shared/model/server-clock'
import { ActionButton, DataTable, EmptyState, RecordDrawer, ToolBar, WidgetFrame } from '@/shared/ui'
import { ENTRY_KIND, eventTitle, eventTypeGroups, kindText, kindTone, signatureText, signatureTone, sourceText } from '../model/labels'
import IntegrityBar from './IntegrityBar.vue'
import JournalRecord from './JournalRecord.vue'
import ToneTag from './ToneTag.vue'

const props = defineProps<WidgetProps>()
const { t, te, d } = useI18n()
const drill = useDrillDown()
const size = computed(() => naiveSizeOf(props.density))

// ── отбор ──
const ALL = 'all'
const kind = ref<JournalEntryKind | typeof ALL>(asEntryKind(props.slice.entry_kind) ?? ALL)
const itemDraft = ref('')
const itemId = ref<string | null>(null)
const eventType = ref<string | null>(null)
const applyItem = () => {
  itemId.value = itemDraft.value.trim() || null
}
const filter = computed<JournalFilter>(() => ({
  ...(kind.value !== ALL ? { entry_kind: kind.value } : {}),
  ...(itemId.value ? { item_id: itemId.value } : {}),
  ...(eventType.value ? { event_type: eventType.value } : {}),
}))
const filtered = computed(() => !!itemId.value || !!eventType.value || kind.value !== (asEntryKind(props.slice.entry_kind) ?? ALL))
function reset(): void {
  kind.value = asEntryKind(props.slice.entry_kind) ?? ALL
  itemDraft.value = ''
  itemId.value = eventType.value = null
}
const typeOptions = computed(() => eventTypeGroups(t, te))

// ── данные ──
const entries = useJournalEntries(filter)
const items = computed(() => entries.items.value ?? [])

const integrityQ = useIntegrity()
const integrity = computed(() => integrityQ.data.value?.data)
// Свежесть — по часам сервера (Ant-Now), а не браузера (AD-46).
const serverNow = useServerNow()
const status = computed(() => (integrity.value ? effectiveIntegrity(integrity.value, serverNow.value) : null))
const reportsQ = useVerifierReports()
const checkedUpTo = computed(() => reportsQ.data.value?.data?.items?.[0]?.checked_up_to_seq ?? null)
const headQ = useJournalHead()
const head = computed(() => headQ.data.value?.data ?? null)

const frameState = computed(() => (status.value === 'violated' ? 'defect_indication' : 'normal'))

// ── окно записи ──
const selectedSeq = ref<number | null>(null)
const selectedRow = computed(() => (selectedSeq.value == null ? null : (items.value.find((e) => e.seq === selectedSeq.value) ?? null)))
// Записи нет среди загруженных (переход по основанию) — читаем её по номеру.
const readQ = useJournalEntry(computed(() => (selectedSeq.value != null && !selectedRow.value ? selectedSeq.value : null)))
const shown = computed(() => selectedRow.value ?? (readQ.data.value?.data?.seq === selectedSeq.value ? readQ.data.value?.data : null) ?? null)
const seqByEventId = computed(() => new Map(items.value.map((e) => [e.event_id, e.seq])))

const time = (iso: string) => d(new Date(iso), 'dateTimeSec')
/** occurred_at показываем мелко, только если отличается от времени записи. */
const occurredDiffers = (e: { occurred_at: string; recorded_at: string }) => Date.parse(e.occurred_at) !== Date.parse(e.recorded_at)
const titleOf = (type: string) => eventTitle(type) ?? `UNKNOWN(${type})`
const itemShort = (id: string) => (id.includes(':') ? id.slice(id.lastIndexOf(':') + 1) : id)
const openItem = (id: string) => drill.open({ entity: 'item', id })
</script>

<template>
  <WidgetFrame
    :title-key="titleKey"
    :density="density"
    :mode="entries.mode.value"
    :state="frameState"
    :loading="entries.isPending.value && !entries.items.value"
    :error="entries.items.value ? undefined : entries.error.value"
    :empty="false"
    :data-widget="widgetId"
  >
    <div class="journal ant-box" data-testid="journal">
      <IntegrityBar
        :status="status"
        :checked-at="integrity?.checked_at ?? null"
        :report-ref="integrity?.report_ref ?? null"
        :checked-up-to="checkedUpTo"
        :head="head"
        :head-error="!head && !!headQ.error.value"
      />

      <ToolBar>
        <NRadioGroup v-model:value="kind" :size="size" data-filter="kind" :aria-label="t('widgets.journal.filter.kind')">
          <NRadioButton :value="ALL" data-kind="all">{{ t('widgets.journal.filter.all') }}</NRadioButton>
          <NRadioButton v-for="k in JOURNAL_ENTRY_KINDS" :key="k" :value="k" :data-kind="k">{{ t(ENTRY_KIND[k].filterKey) }}</NRadioButton>
        </NRadioGroup>
        <NInput
          v-model:value="itemDraft"
          class="f-item"
          :size="size"
          clearable
          :placeholder="t('widgets.journal.filter.item')"
          data-filter="item"
          @change="applyItem"
          @clear="(itemDraft = ''), applyItem()"
          @keydown.enter="applyItem"
        />
        <NSelect
          v-model:value="eventType"
          class="f-type"
          :size="size"
          clearable
          filterable
          :options="typeOptions"
          :placeholder="t('widgets.journal.filter.eventType')"
          data-filter="event-type"
        />
        <template #end>
          <NButton v-if="filtered" :size="size" quaternary data-action="reset" @click="reset">{{ t('common.actions.reset') }}</NButton>
          <span class="muted" data-testid="total">{{ t('widgets.journal.total', { n: items.length }) }}</span>
        </template>
      </ToolBar>

      <EmptyState v-if="entries.items.value && !items.length" compact :title="t(filtered || kind !== ALL ? 'widgets.journal.emptyFiltered' : 'widgets.journal.empty')" />
      <DataTable v-else-if="items.length" class="table" :caption="t(titleKey)">
        <colgroup>
          <col class="c-seq" />
          <col class="c-time" />
          <col class="c-kind" />
          <col class="c-event" />
          <col class="c-item" />
          <col class="c-source" />
          <col class="c-sig" />
          <col class="c-ca" />
        </colgroup>
        <thead>
          <tr>
            <th scope="col"><span class="ant-ellipsis">{{ t('widgets.journal.col.seq') }}</span></th>
            <th scope="col"><span class="ant-ellipsis" :title="t('widgets.journal.col.time')">{{ t('widgets.journal.col.time') }}</span></th>
            <th scope="col"><span class="ant-ellipsis" :title="t('widgets.journal.col.kind')">{{ t('widgets.journal.col.kind') }}</span></th>
            <th scope="col"><span class="ant-ellipsis" :title="t('widgets.journal.col.event')">{{ t('widgets.journal.col.event') }}</span></th>
            <th scope="col"><span class="ant-ellipsis" :title="t('widgets.journal.col.item')">{{ t('widgets.journal.col.item') }}</span></th>
            <th scope="col"><span class="ant-ellipsis" :title="t('widgets.journal.col.source')">{{ t('widgets.journal.col.source') }}</span></th>
            <th scope="col"><span class="ant-ellipsis" :title="t('widgets.journal.col.signature')">{{ t('widgets.journal.col.signature') }}</span></th>
            <th scope="col"><span class="ant-ellipsis" :title="t('widgets.journal.col.ca')">{{ t('widgets.journal.col.ca') }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="e in items"
            :key="e.seq"
            class="row"
            tabindex="0"
            :data-seq="e.seq"
            :data-kind="e.entry_kind"
            :data-signature="e.signature_status"
            :aria-selected="e.seq === selectedSeq"
            @click="selectedSeq = e.seq"
            @keydown.enter.prevent="selectedSeq = e.seq"
          >
            <td class="ant-box num"><span class="ant-mono ant-ellipsis" :title="String(e.seq)">{{ e.seq }}</span></td>
            <td class="ant-box">
              <span class="ant-wrap">{{ time(e.recorded_at) }}</span>
              <span v-if="occurredDiffers(e)" class="muted ant-wrap">{{ t('widgets.journal.occurred', { time: time(e.occurred_at) }) }}</span>
            </td>
            <td class="ant-box"><ToneTag :label="kindText(e.entry_kind, t)" :tone="kindTone(e.entry_kind)" /></td>
            <td class="ant-box">
              <span class="ant-clamp-2" :title="titleOf(e.event_type)">{{ titleOf(e.event_type) }}</span>
              <span class="muted ant-mono ant-ellipsis" :title="e.event_type">{{ e.event_type }}</span>
            </td>
            <td class="ant-box">
              <button v-if="e.item_id" type="button" class="link ant-mono ant-ellipsis" :title="e.item_id" data-action="open-item" @click.stop="openItem(e.item_id)">
                {{ itemShort(e.item_id) }}
              </button>
              <span v-else class="muted">—</span>
            </td>
            <td class="ant-box">
              <span class="ant-ellipsis" :title="sourceText(e, t)">{{ sourceText(e, t) }}</span>
              <span class="muted ant-mono ant-ellipsis" :title="e.source_id">{{ e.source_id }}</span>
            </td>
            <td class="ant-box">
              <ToneTag :label="signatureText(e.signature_status, t)" :tone="signatureTone(e.signature_status)" />
              <span v-if="e.signers.length" class="muted ant-mono ant-ellipsis" :title="e.signers.join(', ')">{{ e.signers.join(', ') }}</span>
            </td>
            <td class="ant-box">
              <span v-if="e.ca_ref" class="ant-mono ant-ellipsis" :title="e.ca_ref">{{ e.ca_ref }}</span>
              <span v-else class="muted">—</span>
            </td>
          </tr>
        </tbody>
      </DataTable>

      <div v-if="items.length" class="more">
        <ActionButton v-if="entries.hasMore.value" :size="size" :loading="entries.loadingMore.value" data-action="more" :label="t('widgets.journal.more')" @click="entries.loadMore()" />
        <span v-else class="muted">{{ t('widgets.journal.allShown') }}</span>
      </div>
    </div>

    <RecordDrawer
      :show="selectedSeq != null"
      :kind-label="t('widgets.journal.record.kind')"
      :number="selectedSeq != null ? t('widgets.journal.record.number', { seq: selectedSeq }) : ''"
      :subtitle="shown ? titleOf(shown.event_type) : ''"
      :loading="!shown && readQ.isFetching.value"
      data-record="journal-entry"
      @close="selectedSeq = null"
    >
      <template v-if="shown" #status>
        <ToneTag :label="kindText(shown.entry_kind, t)" :tone="kindTone(shown.entry_kind)" />
        <ToneTag :label="signatureText(shown.signature_status, t)" :tone="signatureTone(shown.signature_status)" />
      </template>
      <JournalRecord v-if="shown" :entry="shown" :seq-by-event-id="seqByEventId" @open-seq="(s) => (selectedSeq = s)" @open-item="openItem" />
      <EmptyState v-else-if="!readQ.isFetching.value" compact :title="t('widgets.journal.record.unavailable')" />
    </RecordDrawer>
  </WidgetFrame>
</template>

<style scoped>
.journal {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

/* Д-78: без горизонтальной прокрутки — столбцы делят ширину, текст сжимается. */
.table :deep(table) {
  table-layout: fixed;
}

.c-seq {
  width: 7%;
}

.c-time {
  width: 13%;
}

.c-kind {
  width: 12%;
}

.c-event {
  width: 22%;
}

.c-item {
  width: 10%;
}

.c-source {
  width: 13%;
}

.c-sig {
  width: 15%;
}

.c-ca {
  width: 8%;
}

.row {
  cursor: pointer;
}

td > span,
td > button {
  display: block;
  max-width: 100%;
}

td > .tone-tag {
  display: inline-flex;
}

.num {
  font-variant-numeric: tabular-nums;
}

.muted {
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.f-item {
  width: 16rem;
}

.f-type {
  width: 20rem;
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

.link:hover {
  text-decoration: underline;
}

.more {
  display: flex;
  justify-content: center;
}
</style>
