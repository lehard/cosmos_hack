<script setup lang="ts">
/**
 * Партнёры и выписки (эпик 41; FR-131, FR-132, AD-19): партнёр — код
 * предприятия, корни доверия из нашего акта регистрации, канал; выписка —
 * направление, партнёр, что в ней (материал, партия или изделие, плавка) и
 * статус происхождения: «подтверждено» / «подтверждено сервером отправителя» /
 * «не подтверждено». Строка — кнопка: открывает окно записи.
 */
import { useI18n } from 'vue-i18n'
import { ORIGIN_TONE, type Partner, type PassportExtract } from '@/entities/federation'
import { statusPalette } from '@/shared/api/generated/statuses'
import { DataTable, EmptyState, SectionPanel } from '@/shared/ui'

defineProps<{ partners: Partner[]; extracts: PassportExtract[] }>()
const emit = defineEmits<{ 'open-extract': [digest: string]; 'open-partner': [code: string] }>()
const { t, d } = useI18n()

const time = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'dateTime') : '—')
const originText = (e: PassportExtract) => t(`widgets.federation.origin.${e.origin_status}`)
const ackText = (e: PassportExtract) => (e.acknowledged == null ? t('widgets.federation.ackPending') : e.acknowledged ? t('widgets.federation.ackReceived') : t('widgets.federation.ackRejected'))
</script>

<template>
  <div class="federation-view ant-box" data-testid="federation">
    <p class="hint ant-wrap">{{ t('widgets.federation.hint') }}</p>

    <SectionPanel :title="t('widgets.federation.extracts')" variant="plain" :padded="false">
      <EmptyState v-if="!extracts.length" compact :title="t('widgets.federation.noExtracts')" />
      <DataTable v-else>
        <thead>
          <tr>
            <th>{{ t('widgets.federation.direction') }}</th>
            <th>{{ t('widgets.federation.partner') }}</th>
            <th>{{ t('widgets.federation.subject') }}</th>
            <th>{{ t('widgets.federation.heat') }}</th>
            <th>{{ t('widgets.federation.originCol') }}</th>
            <th>{{ t('widgets.federation.at') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="e in extracts"
            :key="e.direction + e.extract_digest"
            class="row"
            :data-digest="e.extract_digest"
            :data-origin="e.origin_status"
            tabindex="0"
            @click="emit('open-extract', e.extract_digest)"
            @keydown.enter="emit('open-extract', e.extract_digest)"
          >
            <td><span class="ant-ellipsis">{{ t(`widgets.federation.directions.${e.direction}`) }}</span></td>
            <td><span class="ant-ellipsis">{{ e.partner_code }}</span></td>
            <td>
              <button type="button" class="link ant-wrap" data-testid="extract-open" @click.stop="emit('open-extract', e.extract_digest)">{{ e.label || e.subject?.id || e.extract_digest }}</button>
            </td>
            <td><span class="ant-ellipsis">{{ e.heat_no || '—' }}</span></td>
            <td>
              <span v-if="e.direction === 'incoming'" class="tag ant-box" data-testid="extract-origin">
                <span class="dot" :style="{ background: statusPalette[ORIGIN_TONE[e.origin_status] ?? 'neutral'] }" aria-hidden="true" />
                <span class="ant-ellipsis">{{ originText(e) }}</span>
              </span>
              <span v-else class="muted ant-ellipsis">{{ ackText(e) }}</span>
            </td>
            <td><span class="ant-ellipsis">{{ time(e.at) }}</span></td>
          </tr>
        </tbody>
      </DataTable>
    </SectionPanel>

    <SectionPanel :title="t('widgets.federation.partners')" variant="plain" :padded="false">
      <EmptyState v-if="!partners.length" compact :title="t('widgets.federation.noPartners')" />
      <DataTable v-else>
        <thead>
          <tr>
            <th>{{ t('widgets.federation.partner') }}</th>
            <th>{{ t('widgets.federation.code') }}</th>
            <th>{{ t('widgets.federation.rootsCol') }}</th>
            <th>{{ t('widgets.federation.channel') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="p in partners"
            :key="p.partner_code"
            class="row"
            :data-partner="p.partner_code"
            tabindex="0"
            @click="emit('open-partner', p.partner_code)"
            @keydown.enter="emit('open-partner', p.partner_code)"
          >
            <td>
              <button type="button" class="link ant-wrap" data-testid="partner-open" @click.stop="emit('open-partner', p.partner_code)">{{ p.name }}</button>
            </td>
            <td><span class="mono ant-ellipsis">{{ p.partner_code }}</span></td>
            <td><span class="ant-ellipsis">{{ p.root_fingerprints.length }}</span></td>
            <td><span class="ant-ellipsis">{{ t(`widgets.federation.channels.${p.channel}`) }}</span></td>
          </tr>
        </tbody>
      </DataTable>
    </SectionPanel>
  </div>
</template>

<style scoped>
.federation-view {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

.hint,
.muted {
  margin: 0;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}

.row {
  cursor: pointer;
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

.mono {
  font-family: var(--ant-font-mono, monospace);
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--ant-accent, inherit);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
</style>
