<script setup lang="ts">
/**
 * Содержимое правого окна записи журнала (Д-70; AD-2, AD-44; FR-68, FR-140):
 * тип и версия схемы, вид, поток, изделие, четыре времени (AD-37), источник и
 * класс происхождения, подпись с пояснением и подписантами, цепочка и CA-‹n›,
 * связи (причина, цепочка обработки, исправляемая запись, основание) и
 * содержимое data — «ключ — значение», сложное — свёрнутым JSON.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { JournalEntryView } from '@/entities/integrity'
import { KeyValue, KeyValueList, SectionPanel } from '@/shared/ui'
import { chainText, dataRows, eventTitle, kindText, kindTone, provenanceText, signatureText, signatureTone, SIGNATURE, sourceText } from '../model/labels'
import ToneTag from './ToneTag.vue'

const props = defineProps<{
  entry: JournalEntryView
  /** Загруженные записи: event_id → seq (переход по связи к записи на экране). */
  seqByEventId: ReadonlyMap<string, number>
}>()
const emit = defineEmits<{ 'open-seq': [seq: number]; 'open-item': [id: string] }>()
const { t, d } = useI18n()

const time = (iso: string) => d(new Date(iso), 'dateTimeSec')
const title = computed(() => eventTitle(props.entry.event_type))
const sigHint = computed(() => {
  const s = SIGNATURE[props.entry.signature_status]
  return s ? t(s.hintKey) : ''
})
/** Краткое содержание от сервера (data.summary) — наверху окна, в списке содержимого не повторяется. */
const summary = computed(() => (typeof props.entry.data?.summary === 'string' ? props.entry.data.summary : null))
const rows = computed(() => dataRows(props.entry.data).filter((r) => !(summary.value && r.key === 'summary')))
/** Связи по event_id: если запись загружена — переход, иначе — текст для копирования. */
const links = computed(() =>
  [
    { key: 'causation', id: props.entry.causation_id },
    { key: 'correlation', id: props.entry.correlation_id },
    { key: 'corrects', id: props.entry.corrects ?? null },
  ].filter((l): l is { key: string; id: string } => !!l.id),
)
</script>

<template>
  <div class="journal-record ant-box" data-testid="journal-record" :data-seq="entry.seq">
    <SectionPanel variant="plain">
      <KeyValueList>
        <KeyValue v-if="summary" :label="t('widgets.journal.record.summary')" data-testid="record-summary">{{ summary }}</KeyValue>
        <KeyValue :label="t('widgets.journal.record.type')">
          <span class="ant-wrap">{{ title ?? `UNKNOWN(${entry.event_type})` }}</span>
          <span class="muted ant-mono ant-wrap">{{ entry.event_type }} · {{ t('widgets.journal.record.schema', { v: entry.schema_version }) }}</span>
        </KeyValue>
        <KeyValue :label="t('widgets.journal.record.entryKind')">
          <ToneTag :label="kindText(entry.entry_kind, t)" :tone="kindTone(entry.entry_kind)" />
        </KeyValue>
        <KeyValue :label="t('widgets.journal.record.stream')" :value="entry.stream" mono />
        <KeyValue :label="t('widgets.journal.record.item')">
          <button v-if="entry.item_id" type="button" class="link ant-mono ant-wrap" data-action="open-item" @click="emit('open-item', entry.item_id)">{{ entry.item_id }}</button>
          <template v-else>—</template>
        </KeyValue>
        <KeyValue v-if="entry.run_id" :label="t('widgets.journal.record.run')" :value="entry.run_id" mono />
        <KeyValue :label="t('widgets.journal.record.eventId')" :value="entry.event_id" mono />
      </KeyValueList>
    </SectionPanel>

    <SectionPanel variant="plain" :title="t('widgets.journal.record.times')">
      <KeyValueList>
        <KeyValue :label="t('timeline.timeKind.occurredAt')" :value="time(entry.occurred_at)" />
        <KeyValue :label="t('timeline.timeKind.receivedAt')" :value="time(entry.received_at)" />
        <KeyValue :label="t('timeline.timeKind.recordedAt')" :value="time(entry.recorded_at)" />
        <KeyValue :label="t('widgets.journal.record.committedAt')" :value="time(entry.committed_at)" />
      </KeyValueList>
    </SectionPanel>

    <SectionPanel variant="plain" :title="t('widgets.journal.record.sourceAndSignature')">
      <KeyValueList>
        <KeyValue :label="t('widgets.journal.record.source')" :value="sourceText(entry, t)" />
        <KeyValue :label="t('widgets.journal.record.sourceId')" :value="entry.source_id" mono />
        <KeyValue :label="t('widgets.journal.record.provenance')" :value="provenanceText(entry.provenance_class, t)" />
        <KeyValue :label="t('widgets.journal.record.signature')">
          <ToneTag :label="signatureText(entry.signature_status, t)" :tone="signatureTone(entry.signature_status)" data-testid="record-signature" />
          <span v-if="sigHint" class="muted ant-wrap">{{ sigHint }}</span>
        </KeyValue>
        <KeyValue :label="t('widgets.journal.record.signers')">
          <span v-if="!entry.signers.length" class="muted">{{ t('widgets.journal.signature.noSigners') }}</span>
          <code v-for="s in entry.signers" :key="s" class="signer ant-mono ant-wrap">{{ s }}</code>
        </KeyValue>
        <KeyValue :label="t('widgets.journal.record.chain')" :value="chainText(entry.chain, t)" />
        <KeyValue v-if="entry.ca_ref" :label="t('widgets.journal.record.caRef')" :value="entry.ca_ref" mono />
      </KeyValueList>
    </SectionPanel>

    <SectionPanel variant="plain" :title="t('widgets.journal.record.links')">
      <KeyValueList>
        <KeyValue v-for="l in links" :key="l.key" :label="t(`widgets.journal.record.${l.key}`)">
          <span class="ant-mono ant-wrap" :data-link="l.key">{{ l.id }}</span>
          <button
            v-if="seqByEventId.has(l.id) && seqByEventId.get(l.id) !== entry.seq"
            type="button"
            class="link"
            :data-action="`open-${l.key}`"
            @click="emit('open-seq', seqByEventId.get(l.id)!)"
          >
            {{ t('widgets.journal.record.openRecord', { seq: seqByEventId.get(l.id) }) }}
          </button>
        </KeyValue>
        <KeyValue v-if="entry.basis_seq != null" :label="t('widgets.journal.record.basis')">
          <button type="button" class="link" data-action="open-basis" @click="emit('open-seq', entry.basis_seq!)">
            {{ t('widgets.journal.record.openRecord', { seq: entry.basis_seq }) }}
          </button>
        </KeyValue>
      </KeyValueList>
    </SectionPanel>

    <SectionPanel variant="plain" :title="t('widgets.journal.record.data')">
      <p v-if="!entry.data" class="muted ant-wrap" data-testid="data-hidden">{{ t('widgets.journal.record.dataHidden') }}</p>
      <p v-else-if="!rows.length && !summary" class="muted ant-wrap">{{ t('widgets.journal.record.dataEmpty') }}</p>
      <KeyValueList v-else-if="rows.length" data-testid="record-data">
        <KeyValue v-for="r in rows" :key="r.key" :label="r.key">
          <template v-if="r.text !== null">{{ r.text }}</template>
          <details v-else class="json">
            <summary>{{ t('common.actions.expand') }}</summary>
            <pre class="ant-mono ant-wrap">{{ r.json }}</pre>
          </details>
        </KeyValue>
      </KeyValueList>
    </SectionPanel>
  </div>
</template>

<style scoped>
.journal-record {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.muted {
  display: block;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.signer {
  display: block;
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

.json summary {
  color: var(--ant-accent);
  font-size: var(--ant-fs-meta);
  cursor: pointer;
}

.json pre {
  margin: var(--ant-space-1) 0 0;
  padding: var(--ant-space-2);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface-subtle);
  font-size: var(--ant-fs-meta);
  white-space: pre-wrap;
}
</style>
