<script setup lang="ts">
/**
 * Журнал критических действий — представление стола Аудитора ИБ (только
 * чтение; FR-79, FR-106; AD-28, AD-44; кейс Т3): CA-‹n› с группой, действием,
 * объектом, «было → стало», кем, по какому полномочию и клейму; отмены — только
 * новой записью со ссылкой. Выбранная запись — с основаниями и связью с
 * основным журналом (запись и обязательство).
 */
import { useI18n } from 'vue-i18n'
import { CA_GROUPS, type CriticalAction, type CriticalActionCaGroup } from '@/entities/integrity'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { DataTable, EmptyState } from '@/shared/ui'

withDefaults(
  defineProps<{
    actions: CriticalAction[]
    group: CriticalActionCaGroup | null
    selected?: string | null
    detail?: CriticalAction | null
    detailError?: unknown
    density?: Density
  }>(),
  { selected: null, detail: null, detailError: undefined, density: 'compact' },
)
const emit = defineEmits<{ 'update:group': [group: CriticalActionCaGroup | null]; select: [caRef: string] }>()
const { t, d } = useI18n()
const problemText = useProblemText()

const time = (iso: string) => d(new Date(iso), 'dateTime')
const groupText = (g: string) => t(`widgets.audit.caGroup.${codeToKey(g)}`)
const groupOptions = CA_GROUPS.map((g) => ({ label: groupText(g), value: g }))
</script>

<template>
  <div class="ca-log" :class="`density-${density}`" data-testid="critical-actions-log">
    <p class="muted">{{ t('widgets.audit.readOnly') }} · {{ t('audit.cancelOnlyByNewRecord') }}</p>
    <label class="filter">
      <span>{{ t('widgets.audit.group') }}</span>
      <select :value="group ?? ''" data-testid="group" @change="(e) => emit('update:group', ((e.target as HTMLSelectElement).value || null) as CriticalActionCaGroup | null)">
        <option value="">{{ t('widgets.audit.allGroups') }}</option>
        <option v-for="o in groupOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
      </select>
    </label>
    <EmptyState v-if="!actions.length" compact :title="t('empty.noRecords')" />
    <DataTable v-else>
      <thead>
        <tr>
          <th>{{ t('audit.criticalAction') }}</th>
          <th>{{ t('audit.record.time') }}</th>
          <th>{{ t('audit.record.action') }}</th>
          <th>{{ t('audit.record.object') }}</th>
          <th>{{ t('audit.record.beforeAfter') }}</th>
          <th>{{ t('audit.record.who') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="a in actions"
          :key="a.ca_ref"
          :data-ca="a.ca_ref"
          :data-cancelled="a.cancelled_by ? 'true' : undefined"
          :aria-selected="a.ca_ref === selected"
          tabindex="0"
          @click="emit('select', a.ca_ref)"
          @keydown.enter.prevent="emit('select', a.ca_ref)"
        >
          <td>
            <strong>{{ a.ca_ref }}</strong>
            <div class="muted">{{ groupText(a.ca_group) }}</div>
          </td>
          <td>{{ time(a.recorded_at) }}</td>
          <td><code>{{ a.action_type }}</code></td>
          <td>{{ a.object.entity }}:{{ a.object.id }}</td>
          <td>
            {{ a.before || '—' }} → {{ a.after || '—' }}
            <div v-if="a.cancels" class="muted">{{ t('audit.cancels', { caId: a.cancels }) }}</div>
            <div v-if="a.cancelled_by" class="warn">{{ t('widgets.audit.cancelledBy', { caId: a.cancelled_by }) }}<template v-if="a.cancel_reason">: {{ a.cancel_reason }}</template></div>
          </td>
          <td>
            {{ a.actor_id ?? '—' }}
            <div v-if="a.authority_id" class="muted">{{ t('audit.record.authority') }}: {{ a.authority_id }}</div>
            <div v-if="a.stamp_id" class="muted">{{ t('audit.record.stamp') }}: {{ a.stamp_id }}</div>
          </td>
        </tr>
      </tbody>
    </DataTable>

    <section v-if="selected" class="card" data-testid="detail">
      <p v-if="detailError" class="error">{{ problemText(detailError) }}</p>
      <dl v-else-if="detail" class="facts">
        <dt>{{ t('audit.criticalAction') }}</dt>
        <dd>{{ t('audit.caNumber', { caId: detail.ca_ref }) }} (№ {{ detail.ca_no }})</dd>
        <dt>{{ t('audit.record.basis') }}</dt>
        <dd>
          <span v-if="!detail.basis_event_ids.length">—</span>
          <code v-for="id in detail.basis_event_ids" :key="id" class="id">{{ id }}</code>
        </dd>
        <dt>{{ t('widgets.audit.mainEvent') }}</dt>
        <dd><code>{{ detail.main_event_id }}</code></dd>
        <dt>{{ t('widgets.audit.commit') }}</dt>
        <dd><code>{{ detail.main_commit }}</code></dd>
        <dt>{{ t('audit.record.object') }}</dt>
        <dd><code>{{ detail.object_ref }}</code></dd>
        <template v-if="detail.policy_seq !== undefined">
          <dt>{{ t('audit.record.authority') }}</dt>
          <dd>{{ t('widgets.audit.policySeq', { seq: detail.policy_seq }) }}</dd>
        </template>
        <template v-if="detail.attested_by">
          <dt>{{ t('audit.signatureCheck.methodPaper') }}</dt>
          <dd>{{ t('widgets.audit.attestedBy', { who: detail.attested_by }) }}</dd>
        </template>
      </dl>
    </section>
    <p v-else-if="actions.length" class="muted">{{ t('widgets.audit.selectAction') }}</p>
  </div>
</template>

<style scoped>
.ca-log {
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: var(--ant-fs-body);
}

.filter {
  display: flex;
  gap: 8px;
  align-items: center;
  max-width: 420px;
}

.filter span {
  white-space: nowrap;
}

.filter select {
  flex: 1;
  padding: 3px 6px;
  border: 1px solid var(--ant-border-strong);
  border-radius: var(--ant-radius-sm);
  background: var(--ant-surface);
  font: inherit;
}

tbody tr {
  cursor: pointer;
}

tr[aria-selected='true'] {
  background: var(--ant-accent-soft);
}

tr[data-cancelled] td {
  color: var(--ant-text-3);
}

.card {
  padding: 8px;
  border: 1px solid var(--ant-border);
  border-radius: var(--ant-radius-md);
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

.id {
  display: block;
}

code {
  font-family: var(--ant-font-mono);
  font-size: var(--ant-fs-meta);
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.warn,
.error {
  color: var(--ant-status-danger);
}
</style>
