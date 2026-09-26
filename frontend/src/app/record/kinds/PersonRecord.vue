<script setup lang="ts">
/**
 * Окно сотрудника (Д-70): заголовок — имя и подразделение; внутри — профиль
 * (`access.person.read`: учётная запись, роли в областях), пост, на который
 * сотрудник назначен сейчас (строка панели «Посты» — открывается окном поста),
 * и квалификации с аттестациями (`access.qualification.list`, FR-80).
 * Чтения без права у роли — честная ошибка в своём блоке, а не пустой профиль.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { usePerson } from '@/entities/person'
import { PRESENCE_TEXT, usePosts, useQualifications } from '@/entities/workplace'
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
const runParams = computed(() => (runId.value ? { run_id: runId.value } : {}))

const personQ = usePerson(() => props.id, runId)
const person = computed(() => personQ.data.value?.data ?? null)

// Пост сейчас — из панели «Посты» (тот же запрос, что у виджета).
const postsQ = usePosts(runParams)
const posts = computed(() => (postsQ.data.value?.data ?? []).filter((r) => r.assigned?.person_id === props.id))
const displayName = computed(() => person.value?.display_name ?? posts.value[0]?.assigned?.display ?? props.id)

const qualsQ = useQualifications(() => props.id, runId)
const quals = computed(() => qualsQ.data.value?.data ?? [])

const date = (iso: string | null | undefined) => (iso ? d(new Date(iso), 'date') : '—')
const roleText = (role: string) => (te(`roles.${codeToKey(role)}`) ? t(`roles.${codeToKey(role)}`) : role)
const accountText = (s: string) => t(`widgets.admin.access.accountStatus.${s}`)

const openPost = (id: string) => drill.open({ entity: 'workplace', id })
</script>

<template>
  <RecordDrawer
    :show="show"
    :kind-label="t('shell.record.person.kind')"
    :number="displayName"
    :subtitle="person?.org_unit ?? ''"
    :loading="personQ.isPending.value && postsQ.isPending.value"
    data-record="person"
    @close="emit('close')"
  >
    <template v-if="person" #status>
      <span class="ant-ellipsis">{{ accountText(person.account_status) }}</span>
    </template>

    <div class="sections ant-box">
      <SectionPanel :title="t('shell.record.person.profile')" variant="plain" :padded="false" data-testid="person-profile">
        <KeyValueList>
          <KeyValue :label="t('shell.record.person.name')" :value="displayName" />
          <KeyValue :label="t('access.conditionalId')" :value="id" mono />
          <template v-if="person">
            <KeyValue :label="t('shell.record.person.orgUnit')" :value="person.org_unit ?? null" />
            <KeyValue :label="t('access.username')" :value="person.login ?? null" mono />
            <KeyValue :label="t('shell.record.person.account')" :value="accountText(person.account_status)" />
          </template>
        </KeyValueList>
        <EmptyState
          v-if="personQ.error.value"
          compact
          :title="t('shell.record.person.profileUnavailable')"
          :description="problemText(personQ.error.value)"
          data-testid="person-profile-error"
        />
      </SectionPanel>

      <SectionPanel v-if="person" :title="t('shell.record.person.roles')" variant="plain" :padded="false" data-testid="person-roles">
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
        <DataTable v-if="posts.length">
          <thead>
            <tr>
              <th>{{ t('liveMap.posts.station') }}</th>
              <th>{{ t('liveMap.posts.presence') }}</th>
              <th>{{ t('liveMap.posts.currentItem') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in posts" :key="p.workplace_id" class="row" :data-workplace="p.workplace_id" @click="openPost(p.workplace_id)">
              <td>
                <button type="button" class="link ant-wrap" @click.stop="openPost(p.workplace_id)">{{ p.station }}</button>
              </td>
              <td><span class="ant-wrap">{{ t(PRESENCE_TEXT[p.presence]) }}</span></td>
              <td><span class="ant-wrap mono">{{ p.current_item?.label ?? '—' }}</span></td>
            </tr>
          </tbody>
        </DataTable>
        <EmptyState v-else-if="postsQ.error.value" compact :title="t('errors.loadFailed')" :description="problemText(postsQ.error.value)" />
        <EmptyState v-else-if="postsQ.data.value" compact :title="t('shell.record.person.noPost')" />
      </SectionPanel>

      <SectionPanel :title="t('access.qualifications')" variant="plain" :padded="false" data-testid="person-qualifications">
        <EmptyState
          v-if="qualsQ.error.value"
          compact
          :title="t('shell.record.person.qualificationsUnavailable')"
          :description="problemText(qualsQ.error.value)"
        />
        <EmptyState v-else-if="qualsQ.data.value && !quals.length" compact :title="t('shell.record.person.noQualifications')" />
        <DataTable v-else-if="quals.length">
          <thead>
            <tr>
              <th>{{ t('shell.record.person.qualification') }}</th>
              <th>{{ t('common.words.status') }}</th>
              <th>{{ t('shell.record.person.validFrom') }}</th>
              <th>{{ t('access.validUntil') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="q in quals" :key="q.qualification_id" :data-qualification="q.status">
              <td>
                <span class="ant-wrap">{{ q.scope ?? q.qualification_id }}</span>
                <span v-if="q.certificate_ref" class="muted mono ant-wrap">{{ q.certificate_ref }}</span>
              </td>
              <td><span class="ant-wrap">{{ t(`shell.record.person.qualificationStatus.${q.status}`) }}</span></td>
              <td><span class="ant-ellipsis">{{ date(q.valid_from) }}</span></td>
              <td><span class="ant-ellipsis">{{ date(q.valid_until) }}</span></td>
            </tr>
          </tbody>
        </DataTable>
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
