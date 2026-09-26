<script setup lang="ts">
/**
 * Окно сотрудника (Д-70, UI-16): заголовок — имя и подразделение; внутри —
 * карточка `access.person.card`: профиль, роли в областях, посты, на которые
 * сотрудник назначен сейчас (щелчок — окно поста), квалификации со сроками (FR-80).
 * Чтение без права у роли — честная ошибка, а не пустой профиль.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { usePersonCard } from '@/entities/person'
import { useDrillDown } from '@/features/drill-down'
import { codeToKey } from '@/shared/i18n'
import { useProblemText } from '@/shared/i18n/problem'
import { DataTable, EmptyState, KeyValue, KeyValueList, RecordDrawer, SectionPanel } from '@/shared/ui'

const props = defineProps<{ id: string; show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t, te, d } = useI18n()
const route = useRoute()
const drill = useDrillDown()
const problemText = useProblemText()

const runId = computed(() => (typeof route.query.run === 'string' && route.query.run ? route.query.run : undefined))

const cardQ = usePersonCard(() => props.id, runId)
const person = computed(() => cardQ.data.value?.data ?? null)
const displayName = computed(() => person.value?.display_name || props.id)

const date = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'date') : '—')
const roleText = (role: string) => (te(`roles.${codeToKey(role)}`) ? t(`roles.${codeToKey(role)}`) : role)
/** Квалификация по-русски из словаря; нет в словаре — код как есть. */
const qualName = (id: string) => {
  const key = `shell.record.person.qualificationName.${codeToKey(id)}`
  return /^[a-z0-9_]+$/.test(id) && te(key) ? t(key) : id
}
const postRoleText = (role: string | undefined) =>
  role ? t(role === 'quality_inspector' ? 'widgets.shopFloor.shift.inspector' : 'widgets.shopFloor.shift.performer') : '—'

const openPost = (id: string) => drill.open({ entity: 'workplace', id })
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('shell.record.person.kind')"
    :number="displayName"
    :subtitle="person?.org_unit ?? ''"
    :loading="cardQ.isPending.value && !person && !cardQ.error.value"
    data-record="person"
    @close="emit('close')"
  >
    <EmptyState
      v-if="!person && cardQ.error.value"
      compact
      :title="t('shell.record.person.unavailable')"
      :description="problemText(cardQ.error.value)"
      data-testid="person-card-error"
    />
    <div v-else-if="person" class="sections ant-box">
      <SectionPanel :title="t('shell.record.person.profile')" variant="plain" :padded="false" data-testid="person-profile">
        <KeyValueList>
          <KeyValue :label="t('shell.record.person.name')" :value="displayName" />
          <KeyValue :label="t('shell.record.person.orgUnit')" :value="person.org_unit ?? null" />
          <KeyValue :label="t('access.conditionalId')" :value="person.person_id" mono />
        </KeyValueList>
      </SectionPanel>

      <SectionPanel :title="t('shell.record.person.roles')" variant="plain" :padded="false" data-testid="person-roles">
        <DataTable v-if="person.roles.length">
          <thead>
            <tr>
              <th>{{ t('access.role') }}</th>
              <th>{{ t('access.scope') }}</th>
              <th>{{ t('access.validUntil') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in person.roles" :key="`${r.role_id}:${r.scope}`">
              <td><span class="ant-wrap">{{ roleText(r.role_id) }}</span></td>
              <td><span class="ant-wrap mono">{{ r.scope }}</span></td>
              <td><span class="ant-ellipsis">{{ date(r.valid_until) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
        <EmptyState v-else compact :title="t('shell.record.person.noRoles')" />
      </SectionPanel>

      <SectionPanel :title="t('shell.record.person.post')" variant="plain" :padded="false" data-testid="person-post">
        <DataTable v-if="person.posts.length">
          <thead>
            <tr>
              <th>{{ t('liveMap.posts.station') }}</th>
              <th>{{ t('shell.record.workplace.roleOnPost') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="p in person.posts"
              :key="`${p.workplace_id}:${p.assignee_role ?? ''}`"
              class="row"
              :data-workplace="p.workplace_id"
              @click="openPost(p.workplace_id)"
            >
              <td>
                <button type="button" class="link ant-wrap" @click.stop="openPost(p.workplace_id)">{{ p.station }}</button>
              </td>
              <td><span class="ant-wrap">{{ postRoleText(p.assignee_role) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
        <EmptyState v-else compact :title="t('shell.record.person.noPost')" />
      </SectionPanel>

      <SectionPanel :title="t('access.qualifications')" variant="plain" :padded="false" data-testid="person-qualifications">
        <DataTable v-if="person.qualifications.length">
          <thead>
            <tr>
              <th>{{ t('shell.record.person.qualification') }}</th>
              <th>{{ t('common.words.status') }}</th>
              <th>{{ t('shell.record.person.validFrom') }}</th>
              <th>{{ t('access.validUntil') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="q in person.qualifications" :key="q.qualification_id" :data-qualification="q.status">
              <td>
                <span class="ant-wrap">{{ qualName(q.qualification_id) }}</span>
                <span v-if="q.certificate_ref" class="muted mono ant-wrap">{{ q.certificate_ref }}</span>
              </td>
              <td><span class="ant-wrap">{{ t(`shell.record.person.qualificationStatus.${q.status}`) }}</span></td>
              <td><span class="ant-ellipsis">{{ date(q.valid_from) }}</span></td>
              <td><span class="ant-ellipsis">{{ date(q.valid_until) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
        <EmptyState v-else compact :title="t('shell.record.person.noQualifications')" />
      </SectionPanel>
    </div>
  </RecordDrawer>
</template>

<style scoped>
.sections {
  display: flex;
  flex-direction: column;
  gap: var(--ant-space-6);
}

.row {
  cursor: pointer;
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
  display: block;
  color: var(--ant-text-3);
  font-size: var(--ant-fs-meta);
}
</style>
