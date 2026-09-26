<script setup lang="ts">
/**
 * Люди и оборудование — представление (PRD §3a, FR-6, FR-17, FR-80, FR-83,
 * FR-84): на посту — назначенный, на месте ли (СКУД, ключ), допуск к рабочему
 * месту, квалификация; оборудование — работает ли, предупреждения («ресурс
 * инструмента 73/75»), остановки, срок поверки.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NDivider, NEllipsis, NFlex, NList, NListItem, NTag, NText } from 'naive-ui'
import { toolLifeOf } from '@/entities/equipment'
import { PRESENCE_TEXT, presenceTagType } from '@/entities/workplace'
import type { Density } from '@/shared/config/widget'
import { useProblemText } from '@/shared/i18n/problem'
import { codeToKey } from '@/shared/i18n'
import { verificationOf, type EquipmentRow, type PersonRow } from '../model/people'

const props = withDefaults(
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
const { t, d } = useI18n()
const problemText = useProblemText()
const tagSize = computed(() => (props.density === 'large' ? 'medium' : 'small'))
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
  <NFlex vertical :size="12" data-testid="people-equipment">
    <NText depth="3">
      {{ workshopName ? t('widgets.shopFloor.station.workshop', { name: workshopName }) : t('widgets.shopFloor.station.noWorkshop') }}
    </NText>

    <section data-testid="people">
      <NDivider title-placement="left">{{ t('widgets.shopFloor.people.title') }}</NDivider>
      <NText v-if="peopleError && !people" type="error">{{ problemText(peopleError) }}</NText>
      <NText v-else-if="people && !people.length" depth="3">{{ t('empty.noRecords') }}</NText>
      <NList v-else-if="people" :show-divider="true">
        <NListItem v-for="p in people" :key="p.post.workplace_id" :data-workplace="p.post.workplace_id" :data-presence="p.post.presence">
          <NFlex vertical :size="4">
            <NFlex :size="8" align="center" :wrap="true">
              <NText strong><NEllipsis :tooltip="{ width: 360 }">{{ p.post.station }}</NEllipsis></NText>
              <NText>{{ p.post.assigned?.display ?? t('liveMap.posts.notAssigned') }}</NText>
              <NTag :size="tagSize" :bordered="false" :type="presenceTagType(p.post.presence)">{{ t(PRESENCE_TEXT[p.post.presence]) }}</NTag>
            </NFlex>
            <NFlex v-if="p.assignment || p.qualifications.length" :size="6" :wrap="true">
              <NTag v-if="p.assignment" :size="tagSize" :bordered="false" :type="p.assignment.admitted ? 'success' : 'warning'" data-testid="admission">
                {{ t(p.assignment.admitted ? 'widgets.shopFloor.people.admitted' : 'widgets.shopFloor.people.notAdmitted') }}
              </NTag>
              <NTag v-if="p.assignment && !p.assignment.qualification_ok" :size="tagSize" :bordered="false" type="error">
                {{ t('widgets.shopFloor.people.qualificationNotOk') }}
              </NTag>
              <NTag v-for="q in p.qualifications" :key="q.qualification_id" :size="tagSize" :bordered="false" :type="qualType(q.status)" :data-qualification="q.status">
                <NEllipsis :tooltip="{ width: 360 }">
                  {{ t(`widgets.shopFloor.qualification.${q.status}`, { what: q.scope ?? q.qualification_id, date: date(q.valid_until) }) }}
                </NEllipsis>
              </NTag>
            </NFlex>
          </NFlex>
        </NListItem>
      </NList>
    </section>

    <section data-testid="equipment">
      <NDivider title-placement="left">{{ t('common.words.equipment') }}</NDivider>
      <NText v-if="equipmentError && !equipment" type="error">{{ problemText(equipmentError) }}</NText>
      <NText v-else-if="equipment && !equipment.length" depth="3">{{ t('empty.noRecords') }}</NText>
      <NList v-else-if="equipment" :show-divider="true">
        <NListItem v-for="e in equipment" :key="e.id" :data-equipment="e.id">
          <NFlex vertical :size="4">
            <NFlex :size="8" align="center" :wrap="true">
              <NText strong><NEllipsis :tooltip="{ width: 360 }">{{ e.title }}</NEllipsis></NText>
              <template v-if="e.state">
                <NTag :size="tagSize" :bordered="false" :type="e.state.execution === 'running' ? 'success' : e.state.execution === 'unknown' ? 'default' : 'warning'" data-testid="execution">
                  {{ t(`widgets.shopFloor.execution.${e.state.execution}`) }}
                </NTag>
                <NTag v-if="e.state.condition !== 'normal'" :size="tagSize" :bordered="false" :type="e.state.condition === 'fault' ? 'error' : e.state.condition === 'warning' ? 'warning' : 'default'">
                  {{ t(`widgets.shopFloor.condition.${e.state.condition}`) }}
                </NTag>
                <NText v-if="toolLifeOf(e.state)" depth="3" data-testid="tool-life">{{ t('timeline.equipment.toolLife', toolLifeOf(e.state)!) }}</NText>
              </template>
              <NTag :size="tagSize" :bordered="false" :type="verificationText(e).type" data-testid="verification">{{ verificationText(e).text }}</NTag>
            </NFlex>
            <template v-if="e.state">
              <NText v-for="w in e.state.warnings" :key="`${w.kind}-${w.since}`" type="warning" data-testid="equipment-warning">
                <NEllipsis :line-clamp="2" :tooltip="{ width: 360 }">{{ w.text }}</NEllipsis>
              </NText>
            </template>
          </NFlex>
        </NListItem>
      </NList>
    </section>
  </NFlex>
</template>
