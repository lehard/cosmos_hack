<script setup lang="ts">
/**
 * «Мои расследования» (UI-32): карточка на инцидент — что расследуем, сколько
 * изделий в области сейчас и сколько было при открытии, идёт ли расследование.
 * Выбранная карточка — рабочее пространство раздела (область риска, дорожки,
 * гипотезы). Открытые — первыми, свежие — раньше; карточки — полосой над
 * рабочим пространством (на узком экране — одна под другой).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { FACTOR_TEXT } from '@/entities/incident'
import type { IncidentSummary } from '@/shared/api/generated/model'

const props = defineProps<{ incidents: readonly IncidentSummary[]; selected: string | null }>()
const emit = defineEmits<{ select: [incidentId: string] }>()

const { t, d } = useI18n()

const sorted = computed(() =>
  [...props.incidents].sort((a, b) => Number(a.status !== 'open') - Number(b.status !== 'open') || b.opened_at.localeCompare(a.opened_at)),
)
const factor = (i: IncidentSummary) =>
  i.common_factor ? t('riskScope.commonFactor', { factor: `${t(FACTOR_TEXT[i.common_factor.factor])}: ${i.common_factor.value}` }) : null
</script>

<template>
  <ul class="list" data-testid="investigations">
    <li v-for="i in sorted" :key="i.incident_id">
      <button
        type="button"
        class="card"
        :data-incident="i.incident_id"
        :data-status="i.status"
        :aria-current="i.incident_id === selected ? 'true' : undefined"
        @click="emit('select', i.incident_id)"
      >
        <span class="label ant-clamp-2">{{ i.label }}</span>
        <span class="scope">
          <span class="now">{{ t('plural.items', { n: i.size }, i.size) }}</span>
          <span class="muted">{{ t('widgets.analysis.investigations.inScope') }}</span>
        </span>
        <span v-if="i.initial_size !== i.size" class="path muted" data-testid="path">{{ t('widgets.analysis.investigations.path', { from: i.initial_size, to: i.size }) }}</span>
        <span v-if="factor(i)" class="muted ant-ellipsis" :title="factor(i) ?? undefined">{{ factor(i) }}</span>
        <span class="state" data-testid="state">
          {{ t(i.status === 'open' ? 'widgets.analysis.investigations.open' : 'widgets.analysis.investigations.closed') }}
          · {{ t('widgets.analysis.investigations.opened', { time: d(new Date(i.opened_at), 'dateTime') }) }}
        </span>
      </button>
    </li>
  </ul>
</template>

<style scoped>
.list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--ant-space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.card {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
  min-width: 0;
  padding: var(--ant-space-2) var(--ant-space-3);
  border: 1px solid var(--ant-border);
  border-left: 4px solid var(--ant-status-attention);
  border-radius: var(--ant-radius-md);
  background: var(--ant-surface);
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.card[data-status='closed'] {
  border-left-color: var(--ant-status-success);
}

.card:hover {
  background: var(--ant-surface-hover);
}

.card[aria-current='true'] {
  border-color: var(--ant-accent);
  background: var(--ant-accent-soft);
}

.scope {
  display: flex;
  flex-wrap: wrap;
  gap: 0 var(--ant-space-1);
  align-items: baseline;
}

.label {
  font-weight: var(--ant-fw-bold);
}

.now {
  font-size: var(--ant-fs-title);
  font-weight: var(--ant-fw-bold);
}

.muted {
  color: var(--ant-text-3);
}

.path,
.state {
  font-size: var(--ant-fs-meta);
}

.state {
  color: var(--ant-text-2);
}
</style>
