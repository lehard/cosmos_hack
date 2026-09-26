<script setup lang="ts">
/**
 * Люди и оборудование — представление (PRD §3a, FR-6, FR-17, FR-80, FR-83,
 * FR-84): на посту — назначенный, на месте ли (СКУД, ключ), допуск к рабочему
 * месту, квалификация; оборудование — работает ли, предупреждения («ресурс
 * инструмента 73/75»), остановки, срок поверки. Элементы и токены «Главного».
 */
import { useI18n } from 'vue-i18n'
import { NAlert, NTag } from 'naive-ui'
import { toolLifeOf } from '@/entities/equipment'
import { PRESENCE_TEXT, presenceTagType } from '@/entities/workplace'
import type { Density } from '@/shared/config/widget'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { ActionButton, DataTable, EmptyState, SectionPanel } from '@/shared/ui'
import { verificationOf, type EquipmentRow, type PersonRow } from '../model/people'

withDefaults(
  defineProps<{
    workshopName: string | null
    people: readonly PersonRow[] | null
    peopleError?: unknown
    equipment: readonly EquipmentRow[] | null
    equipmentError?: unknown
    density?: Density
  }>(),
  { peopleError: undefined, equipmentError: undefined, density: 'comfortable' },
)
const emit = defineEmits<{
  /** Пост — окно поста: кто назначен, история (Д-70, UI-43). */
  workplace: [workplaceId: string]
  /** Сотрудник — окно сотрудника. */
  person: [personId: string]
  /** Пост без исполнителя — окно назначения с кандидатами по квалификации. */
  assign: [workplaceId: string, station: string]
}>()
const { t, d } = useI18n()
const problemText = useProblemText()
const date = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'date') : '—')

function verificationText(row: EquipmentRow): { text: string; type: 'success' | 'warning' | 'error' | 'default' } {
  const v = verificationOf(row)
  switch (v.kind) {
    case 'valid':
      return { text: t('widgets.shopFloor.verification.validUntil', { date: date(v.until) }), type: 'success' }
    case 'expiring':
      return { text: t('widgets.shopFloor.verification.expiring', { date: date(v.until) }), type: 'warning' }
    case 'expired':
      return { text: t('widgets.shopFloor.verification.expired', { date: date(v.until) }), type: 'error' }
    case 'unusable':
      return { text: t(`widgets.shopFloor.unusable.${codeToKey(v.reason)}`), type: 'error' }
    case 'not_required':
      return { text: t('widgets.shopFloor.verification.notRequired'), type: 'default' }
    case 'unknown':
      return { text: t('widgets.shopFloor.verification.unknown'), type: 'default' }
  }
  return { text: '—', type: 'default' }
}

const qualType = (s: string) => (s === 'valid' ? 'success' : s === 'expiring' ? 'warning' : 'error')
</script>

<template>
  <div class="people-equipment" data-testid="people-equipment">
    <p class="ant-muted ant-wrap">
      {{ workshopName ? t('widgets.shopFloor.station.workshop', { name: workshopName }) : t('widgets.shopFloor.station.noWorkshop') }}
    </p>

    <SectionPanel :title="t('widgets.shopFloor.people.title')" variant="plain" data-testid="people">
      <NAlert v-if="peopleError && !people" type="error" :bordered="false">{{ problemText(peopleError) }}</NAlert>
      <EmptyState v-else-if="people && !people.length" compact :title="t('empty.noRecords')" />
      <DataTable v-else-if="people" :caption="t('widgets.shopFloor.people.title')">
        <thead>
          <tr>
            <th>{{ t('liveMap.posts.station') }}</th>
            <th>{{ t('liveMap.posts.assigned') }}</th>
            <th>{{ t('liveMap.posts.presence') }}</th>
            <th>{{ t('access.qualifications') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in people" :key="p.post.workplace_id" :data-workplace="p.post.workplace_id" :data-presence="p.post.presence">
            <th scope="row">
              <ActionButton text type="primary" :label="p.post.station" data-testid="open-post" @click="emit('workplace', p.post.workplace_id)" />
            </th>
            <td>
              <ActionButton v-if="p.post.assigned" text type="primary" :label="p.post.assigned.display" data-testid="open-person" @click="emit('person', p.post.assigned.person_id)" />
              <template v-else>
                <span class="ant-wrap">{{ t('liveMap.posts.notAssigned') }}</span>
                <ActionButton size="small" type="primary" secondary :label="t('widgets.shopFloor.now.act.unassigned')" data-testid="assign-post" @click="emit('assign', p.post.workplace_id, p.post.station)" />
              </template>
            </td>
            <td>
              <NTag size="small" :bordered="false" :type="presenceTagType(p.post.presence)"><span class="ant-wrap">{{ t(PRESENCE_TEXT[p.post.presence]) }}</span></NTag>
            </td>
            <td>
              <div class="tags">
                <span v-if="!p.assignment && !p.qualifications.length" class="ant-muted">—</span>
                <NTag v-if="p.assignment" size="small" :bordered="false" :type="p.assignment.admitted ? 'success' : 'warning'" data-testid="admission">
                  {{ t(p.assignment.admitted ? 'widgets.shopFloor.people.admitted' : 'widgets.shopFloor.people.notAdmitted') }}
                </NTag>
                <NTag v-if="p.assignment && !p.assignment.qualification_ok" size="small" :bordered="false" type="error">
                  {{ t('widgets.shopFloor.people.qualificationNotOk') }}
                </NTag>
                <NTag v-for="q in p.qualifications" :key="q.qualification_id" size="small" :bordered="false" :type="qualType(q.status)" :data-qualification="q.status">
                  <span class="ant-wrap">{{ t(`widgets.shopFloor.qualification.${q.status}`, { what: q.scope ?? q.qualification_id, date: date(q.valid_until) }) }}</span>
                </NTag>
              </div>
            </td>
          </tr>
        </tbody>
      </DataTable>
    </SectionPanel>

    <SectionPanel :title="t('common.words.equipment')" variant="plain" data-testid="equipment">
      <NAlert v-if="equipmentError && !equipment" type="error" :bordered="false">{{ problemText(equipmentError) }}</NAlert>
      <EmptyState v-else-if="equipment && !equipment.length" compact :title="t('empty.noRecords')" />
      <DataTable v-else-if="equipment" :caption="t('common.words.equipment')">
        <thead>
          <tr>
            <th>{{ t('common.words.equipment') }}</th>
            <th>{{ t('common.words.status') }}</th>
            <th>{{ t('widgets.shopFloor.verification.title') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in equipment" :key="e.id" :data-equipment="e.id">
            <th scope="row">
              <!-- Оборудование поста — окно поста: кто на нём работал (история поста). -->
              <ActionButton
                v-if="e.state?.station_id"
                text
                type="primary"
                :label="e.title"
                :hint="t('widgets.shopFloor.people.equipmentHistory')"
                data-testid="open-equipment-post"
                @click="emit('workplace', e.state!.station_id!)"
              />
              <span v-else class="ant-wrap">{{ e.title }}</span>
              <template v-if="e.state">
                <p v-for="w in e.state.warnings" :key="`${w.kind}-${w.since}`" class="warning ant-clamp-2" :title="w.text" data-testid="equipment-warning">{{ w.text }}</p>
              </template>
            </th>
            <td>
              <div v-if="e.state" class="tags">
                <NTag size="small" :bordered="false" :type="e.state.execution === 'running' ? 'success' : e.state.execution === 'unknown' ? 'default' : 'warning'" data-testid="execution">
                  {{ t(`widgets.shopFloor.execution.${e.state.execution}`) }}
                </NTag>
                <NTag v-if="e.state.condition !== 'normal'" size="small" :bordered="false" :type="e.state.condition === 'fault' ? 'error' : e.state.condition === 'warning' ? 'warning' : 'default'">
                  {{ t(`widgets.shopFloor.condition.${e.state.condition}`) }}
                </NTag>
                <span v-if="toolLifeOf(e.state)" class="ant-muted" data-testid="tool-life">{{ t('timeline.equipment.toolLife', toolLifeOf(e.state)!) }}</span>
              </div>
              <span v-else class="ant-muted">{{ t('empty.noDataUnknown') }}</span>
            </td>
            <td>
              <NTag size="small" :bordered="false" :type="verificationText(e).type" data-testid="verification">
                <span class="ant-wrap">{{ verificationText(e).text }}</span>
              </NTag>
            </td>
          </tr>
        </tbody>
      </DataTable>
    </SectionPanel>
  </div>
</template>

<style scoped>
.people-equipment {
  display: flex;
  flex-direction: column;
  gap: var(--ant-gap);
  min-width: 0;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ant-space-1) var(--ant-space-2);
  align-items: center;
  min-width: 0;
}

.warning {
  color: var(--ant-status-attention-text);
  font-size: var(--ant-fs-meta);
  font-weight: var(--ant-fw-regular);
}
</style>
