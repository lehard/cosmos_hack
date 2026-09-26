<script setup lang="ts">
/**
 * Окно поста (Д-70, UI-16): заголовок — название поста и цех; внутри — что пост
 * знает сейчас (назначен, на месте ли, текущее изделие) и назначения текущей
 * смены — карточка поста `access.workplace.read`; история поста —
 * `access.workplace.history` (назначения и снятия, ключ, допуск, отклонения
 * присутствия; новые сверху, «показать ещё» — следующая страница).
 * Сотрудник и изделие открываются своими окнами. Чтение без права у роли —
 * честная ошибка в своём блоке, а не пустой список.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useEquipmentStates } from '@/entities/equipment'
import { ProcessHoldAction } from '@/features/process-hold'
import { PRESENCE_TEXT, PRESENCE_TONE, useWorkplaceCard, useWorkplaceHistory, type WorkplaceEvent } from '@/entities/workplace'
import { useDrillDown } from '@/features/drill-down'
import { statusPalette } from '@/shared/api/generated/statuses'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, DataTable, EmptyState, KeyValue, KeyValueList, RecordDrawer, SectionPanel } from '@/shared/ui'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t, te, d } = useI18n()
const route = useRoute()
const drill = useDrillDown()
const problemText = useProblemText()

const runId = computed(() => (typeof route.query.run === 'string' && route.query.run ? route.query.run : undefined))

const cardQ = useWorkplaceCard(() => props.id, runId)
const card = computed(() => cardQ.data.value?.data ?? null)

const history = useWorkplaceHistory(() => props.id, runId)

/** Оборудование этого поста (station_id — пост): остановка — по каждому (FR-49). */
const equipmentQ = useEquipmentStates()
const postEquipment = computed(() => (equipmentQ.data.value?.data ?? []).filter((e) => e.station_id === props.id))

/** Основание события: код из словаря — по-русски, текст снятия — как есть. */
function reasonText(reason: string): string {
  const key = `shell.record.workplace.reason.${codeToKey(reason)}`
  return /^[a-z0-9_]+$/.test(reason) && te(key) ? t(key) : reason
}
function eventText(e: WorkplaceEvent): string {
  const key = `shell.record.workplace.kind.${codeToKey(e.kind)}`
  const title = te(key) ? t(key) : e.kind
  return e.reason ? `${title}: ${reasonText(e.reason)}` : title
}
const historyRows = computed(() =>
  (history.items.value ?? []).map((e) => ({
    key: `${e.seq}:${e.kind}`,
    at: d(new Date(e.at), 'dateTime'),
    event: eventText(e),
    personId: e.person_id || null,
    person: e.person_display || e.person_id || '',
  })),
)

/** Цех по имени; код — только если имени нет. */
const workshop = computed(() => card.value?.workshop_name ?? card.value?.workshop ?? null)
const subtitle = computed(() => (workshop.value ? `${t('common.words.workshop')}: ${workshop.value}` : ''))
const qualText = (ok: boolean) => t(ok ? 'shell.record.workplace.qualificationOk' : 'shell.record.workplace.qualificationMissing')
const roleText = (role: string) => t(role === 'quality_inspector' ? 'widgets.shopFloor.shift.inspector' : 'widgets.shopFloor.shift.performer')

const openPerson = (id: string) => drill.open({ entity: 'person', id })
const openItem = (id: string) => drill.open({ entity: 'item', id })
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('common.words.post')"
    :number="card?.station ?? id"
    :subtitle="subtitle"
    :loading="cardQ.isPending.value && !card && !cardQ.error.value"
    data-record="workplace"
    @close="emit('close')"
  >
    <template v-if="card" #status>
      <span class="presence ant-box" :data-presence="card.presence">
        <span class="dot" :style="{ background: statusPalette[PRESENCE_TONE[card.presence]] }" aria-hidden="true" />
        <span class="ant-ellipsis">{{ t(PRESENCE_TEXT[card.presence]) }}</span>
      </span>
    </template>

    <div class="sections ant-box">
      <SectionPanel :title="t('shell.record.workplace.now')" variant="plain" :padded="false" data-testid="workplace-now">
        <KeyValueList v-if="card">
          <KeyValue :label="t('liveMap.posts.station')" :value="card.station" />
          <KeyValue :label="t('common.words.workshop')" :value="workshop" />
          <KeyValue :label="t('liveMap.posts.assigned')">
            <button v-if="card.assigned" type="button" class="link ant-wrap" data-testid="workplace-person" @click="openPerson(card.assigned.person_id)">
              {{ card.assigned.display }}
            </button>
            <span v-else class="muted ant-wrap">{{ t('liveMap.posts.notAssigned') }}</span>
          </KeyValue>
          <KeyValue :label="t('liveMap.posts.presence')" :value="t(PRESENCE_TEXT[card.presence])" />
          <KeyValue :label="t('liveMap.posts.currentItem')">
            <button
              v-if="card.current_item"
              type="button"
              class="link mono ant-wrap"
              data-testid="workplace-item"
              :title="t('common.actions.openPassport')"
              @click="openItem(card.current_item.item_id)"
            >
              {{ card.current_item.label }}
            </button>
            <span v-else class="muted">—</span>
          </KeyValue>
          <KeyValue :label="t('shell.record.workplace.code')" :value="card.workplace_id" mono />
          <KeyValue :label="t('shell.record.workplace.scope')" :value="card.scope ?? null" mono />
        </KeyValueList>
        <EmptyState
          v-else-if="cardQ.error.value"
          compact
          :title="t('shell.record.workplace.unavailable')"
          :description="problemText(cardQ.error.value)"
          data-testid="workplace-card-error"
        />
      </SectionPanel>

      <SectionPanel v-if="card" :title="t('shell.record.workplace.shiftAssignments')" variant="plain" :padded="false" data-testid="workplace-assignments">
        <DataTable v-if="card.assignments.length">
          <thead>
            <tr>
              <th>{{ t('shell.record.workplace.person') }}</th>
              <th>{{ t('shell.record.workplace.roleOnPost') }}</th>
              <th>{{ t('shell.record.workplace.qualification') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in card.assignments" :key="`${a.shift_id}:${a.person_id}:${a.assignee_role}`">
              <td>
                <button type="button" class="link ant-wrap" @click="openPerson(a.person_id)">{{ a.person_display || a.person_id }}</button>
              </td>
              <td><span class="ant-wrap">{{ roleText(a.assignee_role) }}</span></td>
              <td><span class="ant-wrap" :data-qualification-ok="a.qualification_ok">{{ qualText(a.qualification_ok) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
        <EmptyState v-else compact :title="t('shell.record.workplace.noShiftAssignments')" />
      </SectionPanel>

      <SectionPanel
        :title="t('shell.record.workplace.history')"
        :subtitle="t('shell.record.workplace.historyHint')"
        variant="plain"
        :padded="false"
        data-testid="workplace-history"
      >
        <EmptyState
          v-if="history.error.value && !historyRows.length"
          compact
          :title="t('shell.record.workplace.historyUnavailable')"
          :description="problemText(history.error.value)"
          data-testid="workplace-history-error"
        />
        <div v-else-if="history.isPending.value" class="muted note ant-wrap">{{ t('common.state.loading') }}</div>
        <EmptyState v-else-if="!historyRows.length" compact :title="t('shell.record.workplace.noHistory')" />
        <template v-else>
          <DataTable>
            <thead>
              <tr>
                <th>{{ t('common.words.time') }}</th>
                <th>{{ t('shell.record.workplace.event') }}</th>
                <th>{{ t('shell.record.workplace.person') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="h in historyRows" :key="h.key" data-testid="workplace-history-row">
                <td><span class="ant-ellipsis" :title="h.at">{{ h.at }}</span></td>
                <td><span class="ant-wrap">{{ h.event }}</span></td>
                <td>
                  <button v-if="h.personId" type="button" class="link ant-wrap" @click="openPerson(h.personId)">{{ h.person }}</button>
                  <span v-else class="muted">—</span>
                </td>
              </tr>
            </tbody>
          </DataTable>
          <div v-if="history.hasMore.value" class="more ant-box">
            <ActionButton
              size="small"
              :label="t('shell.record.workplace.showMore')"
              :loading="history.loadingMore.value"
              data-testid="workplace-history-more"
              @click="history.loadMore"
            />
          </div>
          <p v-if="history.error.value" class="muted note ant-wrap">{{ problemText(history.error.value) }}</p>
        </template>
      </SectionPanel>
    </div>

    <template v-if="postEquipment.length" #actions>
      <div class="post-equipment ant-box" data-testid="post-equipment">
        <div v-for="e in postEquipment" :key="e.equipment_id" class="equipment-row ant-box">
          <span class="equipment-title ant-ellipsis" :title="e.title">{{ e.title }}</span>
          <ProcessHoldAction :equipment-id="e.equipment_id" :equipment-title="e.title" />
        </div>
      </div>
    </template>
  </RecordDrawer>
</template>

<style scoped>
.post-equipment {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-3);
}

.equipment-row {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-2);
}

.equipment-title {
  font-weight: var(--ant-fw-bold);
}

.sections {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-6);
}

.presence {
  display: inline-flex;
  gap: var(--ant-space-2);
  align-items: center;
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

.mono {
  font-family: var(--ant-font-mono);
}

.muted {
  color: var(--ant-text-3);
}

.more {
  margin-top: var(--ant-space-2);
}

.note {
  margin: var(--ant-space-2) 0 0;
  font-size: var(--ant-fs-meta);
}
</style>
