<script setup lang="ts">
/**
 * История выдачи прав — представление стола Аудитора ИБ (только чтение;
 * FR-79; AD-11, AD-28): что выдано или отозвано, кому, в какой области, кем,
 * была ли вторая подпись независимой стороны, номер критического действия и
 * документ выдачи. Фильтр — по псевдониму сотрудника.
 */
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NInput } from 'naive-ui'
import type { AccessGrantEntry } from '@/entities/policy'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { DataTable, EmptyState } from '@/shared/ui'

const props = withDefaults(defineProps<{ entries: AccessGrantEntry[]; personId: string | null; density?: Density }>(), { density: 'compact' })
const emit = defineEmits<{ 'update:personId': [personId: string | null] }>()
const { t, d } = useI18n()
const time = (iso: string) => d(new Date(iso), 'dateTime')

// Фильтр уходит на сервер по Enter или при уходе с поля — не на каждую букву.
const draft = ref(props.personId ?? '')
watch(
  () => props.personId,
  (v) => (draft.value = v ?? ''),
)
const apply = () => emit('update:personId', draft.value.trim() || null)
</script>

<template>
  <div class="grants" :class="`density-${density}`" data-testid="grants-history">
    <p class="muted">{{ t('widgets.audit.readOnly') }} · {{ t('access.policyChangesLogged') }}</p>
    <label class="filter">
      <span>{{ t('widgets.audit.grants.personFilter') }}</span>
      <NInput v-model:value="draft" size="small" clearable data-testid="person" @keydown.enter.prevent="apply" @blur="apply" @clear="emit('update:personId', null)" />
    </label>
    <EmptyState v-if="!entries.length" compact :title="t('empty.noRecords')" />
    <DataTable v-else>
      <thead>
        <tr>
          <th>{{ t('audit.record.time') }}</th>
          <th>{{ t('audit.record.action') }}</th>
          <th>{{ t('widgets.admin.access.person') }}</th>
          <th>{{ t('access.scope') }}</th>
          <th>{{ t('widgets.audit.grants.by') }}</th>
          <th>{{ t('widgets.audit.grants.secondSignature') }}</th>
          <th>{{ t('audit.criticalAction') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="g in entries" :key="g.event_id" :data-seq="g.seq" :data-action="g.action">
          <td>{{ time(g.at) }}</td>
          <td>
            <strong>{{ t(`widgets.audit.grants.action.${g.action}`) }}</strong>:
            {{ t(`widgets.audit.grants.kind.${codeToKey(g.kind)}`) }} «{{ g.subject_id }}»
          </td>
          <td>{{ g.person_id ?? '—' }}</td>
          <td>{{ g.scope ?? '—' }}</td>
          <td>{{ g.by }}</td>
          <td :class="{ muted: !g.second_signature_by }">{{ g.second_signature_by ?? t('widgets.audit.grants.noSecondSignature') }}</td>
          <td>
            {{ g.ca_ref ?? '—' }}
            <div v-if="g.document_id" class="muted">{{ t('widgets.audit.grants.document', { id: g.document_id }) }}</div>
          </td>
        </tr>
      </tbody>
    </DataTable>
  </div>
</template>

<style scoped>
.grants {
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

tr[data-action='revoked'] strong {
  color: var(--ant-status-danger);
}

.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
